package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

const (
	fieldManager = "kube-directory-reconciler"

	managedAnnotation = "kube-directory-reconciler.openai.com/managed"

	inventoryNamespace = "kube-system"
	inventoryName      = "kube-directory-reconciler-inventory"
	inventoryDataKey   = "resources.json"

	inventoryVersion = 1
)

type BuildObject struct {
	Object   *unstructured.Unstructured
	Metadata *config.PatchMetadata
}

// ResourceID identifies a Kubernetes resource by its API resource,
// namespace, and name. Version is retained for API access but excluded
// from Key(), since multiple API versions can identify the same object.
type ResourceID struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

func (id ResourceID) Key() string {
	return strings.Join([]string{
		id.Group,
		id.Resource,
		id.Namespace,
		id.Name,
	}, "/")
}

func (id ResourceID) String() string {
	if id.Namespace == "" {
		return fmt.Sprintf("%s/%s %s",
			id.Group, id.Resource, id.Name)
	}
	return fmt.Sprintf("%s/%s %s/%s",
		id.Group, id.Resource, id.Namespace, id.Name)
}

// Action describes an operation planned or performed.
type Action struct {
	Operation string     `json:"operation"`
	Resource  ResourceID `json:"resource"`
}

// Report describes the reconciliation run.
// Actions contains planned operations when DryRun is true.
type Report struct {
	DryRun  bool     `json:"dryRun"`
	Actions []Action `json:"actions"`
}

type storedInventory struct {
	Version   int          `json:"version"`
	Resources []ResourceID `json:"resources"`
}

type reconciler struct {
	dynamic    dynamic.Interface
	clientset  kubernetes.Interface
	discovery  discovery.DiscoveryInterface
	cachedDisc discovery.CachedDiscoveryInterface
	mapper     *restmapper.DeferredDiscoveryRESTMapper
	dryRun     bool
}

// Reconcile loads manifestFiles and reconciles them against the cluster
// selected by kubeConfigPath and contextName.
//
// Each file can contain YAML, JSON, multiple YAML documents, or a
// Kubernetes List. contextName may be empty to use the kubeconfig's
// current context.
//
// dryRun validates proposed apply and delete operations using Kubernetes
// server-side dry-run. It does not persist inventory changes.
//
// Only resources recorded in this reconciler's previous inventory can
// be pruned. Run only one reconciliation at a time per context and
// inventory. Separate manifest sets require separate inventory identities.
func Reconcile(
	ctx context.Context,
	sl *statusline.StatusLine,
	kubeConfigPath string,
	contextName string,
	manifestFiles []string,
	dryRun bool,
) (Report, error) {
	report := Report{
		DryRun:  dryRun,
		Actions: []Action{},
	}

	if strings.TrimSpace(kubeConfigPath) == "" {
		return report, fmt.Errorf("kubeConfigPath must not be empty")
	}

	if len(manifestFiles) == 0 {
		return report, fmt.Errorf(
			"manifestFiles is empty; refusing to prune all previously managed resources",
		)
	}

	for _, path := range manifestFiles {
		if strings.TrimSpace(path) == "" {
			return report, fmt.Errorf("manifestFiles contains an empty path")
		}
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeConfigPath,
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, overrides,
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return report, fmt.Errorf("load kubeconfig: %w", err)
	}

	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return report, fmt.Errorf("create dynamic client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return report, fmt.Errorf("create Kubernetes clientset: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return report, fmt.Errorf("create discovery client: %w", err)
	}

	cachedDiscovery := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	r := &reconciler{
		dynamic:    dyn,
		clientset:  clientset,
		discovery:  discoveryClient,
		cachedDisc: cachedDiscovery,
		mapper:     mapper,
		dryRun:     dryRun,
	}

	// Parse every file before changing the cluster.
	desired, err := loadManifests(sl, manifestFiles)
	if err != nil {
		return report, fmt.Errorf("load manifests: %w", err)
	}

	// Validate duplicate manifest identities without requiring the
	// resource types to already exist in cluster discovery.
	rawIDs := make(map[string]struct{}, len(desired))
	for _, buildObj := range desired {
		obj := buildObj.Object
		key := rawIdentity(obj)
		if sl != nil {
			sl.Emit("checking %s", key)
		}

		if _, exists := rawIDs[key]; exists {
			return report, fmt.Errorf("duplicate manifest resource: %s",
				key)
		}
		rawIDs[key] = struct{}{}

		if isInventoryConfigMap(obj) {
			return report, fmt.Errorf(
				"manifest collides with reserved inventory ConfigMap %s/%s",
				inventoryNamespace, inventoryName,
			)
		}
	}

	previous, err := r.loadInventory(ctx)
	if err != nil {
		return report, fmt.Errorf("load previous inventory: %w", err)
	}

	// Apply prerequisites first. CRDs must be installed before their
	// custom resources can be resolved by discovery.
	sort.SliceStable(desired, func(i, j int) bool {
		return applyRank(desired[i].Object) < applyRank(desired[j].Object)
	})

	current := make(map[string]ResourceID, len(desired))

	for _, buildObj := range desired {
		obj := buildObj.Object
		// A CRD may have been installed earlier in this same run.
		id, err := r.identity(buildObj)
		if err != nil {
			return report, fmt.Errorf(
				"resolve %s %s: %w",
				obj.GetKind(), obj.GetName(), err,
			)
		}

		action, err := r.apply(ctx, obj, id)
		if err != nil {
			return report, fmt.Errorf("apply %s: %w", id, err)
		}
		report.Actions = append(report.Actions, action)
		current[id.Key()] = id

		if obj.GetAPIVersion() == "apiextensions.k8s.io/v1" &&
			obj.GetKind() == "CustomResourceDefinition" {
			r.mapper.Reset()
		}
	}

	// Prune resources that were previously managed but are no longer
	// present in the desired inventory.
	var stale []ResourceID
	for _, old := range previous {
		if _, exists := current[old.Key()]; !exists {
			stale = append(stale, old)
		}
	}

	sort.SliceStable(stale, func(i, j int) bool {
		return deleteRank(stale[i]) > deleteRank(stale[j])
	})

	for _, id := range stale {
		action, err := r.prune(ctx, id)
		if err != nil {
			return report, fmt.Errorf("prune %s: %w", id, err)
		}
		if action != nil {
			report.Actions = append(report.Actions, *action)
		}
	}

	// A dry-run never writes the inventory.
	if dryRun {
		return report, nil
	}

	newInventory := make([]ResourceID, 0, len(current))
	for _, id := range current {
		newInventory = append(newInventory, id)
	}
	sort.Slice(newInventory, func(i, j int) bool {
		return newInventory[i].Key() < newInventory[j].Key()
	})

	if err := r.saveInventory(ctx, newInventory); err != nil {
		return report, fmt.Errorf("save inventory: %w", err)
	}

	return report, nil
}

func rawIdentity(obj *unstructured.Unstructured) string {
	return strings.Join([]string{
		obj.GetAPIVersion(),
		obj.GetKind(),
		obj.GetNamespace(),
		obj.GetName(),
	}, "/")
}

func isInventoryConfigMap(obj *unstructured.Unstructured) bool {
	return obj.GetAPIVersion() == "v1" &&
		obj.GetKind() == "ConfigMap" &&
		obj.GetNamespace() == inventoryNamespace &&
		obj.GetName() == inventoryName
}

func applyRank(obj *unstructured.Unstructured) int {
	switch {
	case obj.GetKind() == "Namespace" &&
		obj.GetAPIVersion() == "v1":
		return 0
	case obj.GetKind() == "CustomResourceDefinition" &&
		obj.GetAPIVersion() == "apiextensions.k8s.io/v1":
		return 1
	default:
		return 2
	}
}

func deleteRank(id ResourceID) int {
	switch {
	case id.Group == "apiextensions.k8s.io" &&
		id.Resource == "customresourcedefinitions":
		return 0
	case id.Group == "" && id.Resource == "namespaces":
		return 1
	default:
		return 2
	}
}

func loadManifests(sl *statusline.StatusLine, paths []string) ([]*BuildObject, error) {
	var result []*BuildObject

	for _, path := range paths {
		if sl != nil {
			sl.Emit("examining %s...", path)
		}
		noExt := strings.TrimSuffix(path, ".yaml")
		metaFile := fmt.Sprintf("%s.meta", noExt)

		patchMeta := &config.PatchMetadata{}
		if _, err := os.Stat(metaFile); err == nil {
			mBytes, err := os.ReadFile(metaFile)
			if err != nil {
				return nil, fmt.Errorf("unable to read patch metadata: %w", err)
			}

			if err = yaml.Unmarshal(mBytes, patchMeta); err != nil {
				return nil, fmt.Errorf("unable to unmarshal patch metadata: %w", err)
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %q: %w", path, err)
		}

		decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
		docNumber := 0

		for {
			var raw map[string]interface{}
			err := decoder.Decode(&raw)

			if err == io.EOF {
				break
			}
			if err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("decode %q: %w", path, err)
			}

			docNumber++
			if len(raw) == 0 {
				continue
			}

			obj := &unstructured.Unstructured{Object: raw}

			if obj.GetKind() == "List" {
				items, found, err := unstructured.NestedSlice(
					obj.Object, "items",
				)
				if err != nil || !found {
					_ = f.Close()
					return nil, fmt.Errorf(
						"%s document %d: invalid Kubernetes List",
						path, docNumber,
					)
				}

				for i, item := range items {
					itemMap, ok := item.(map[string]interface{})
					if !ok {
						_ = f.Close()
						return nil, fmt.Errorf(
							"%s document %d item %d is not an object",
							path, docNumber, i,
						)
					}

					itemObj := &unstructured.Unstructured{
						Object: itemMap,
					}
					if err := validateObject(itemObj); err != nil {
						_ = f.Close()
						return nil, fmt.Errorf(
							"%s document %d item %d: %w",
							path, docNumber, i, err,
						)
					}

					bo := &BuildObject{
						Object:   itemObj,
						Metadata: patchMeta,
					}
					result = append(result, bo)
				}
				continue
			}

			if err := validateObject(obj); err != nil {
				_ = f.Close()
				return nil, fmt.Errorf(
					"%s document %d: %w", path, docNumber, err,
				)
			}
			bo := &BuildObject{
				Object:   obj,
				Metadata: patchMeta,
			}
			result = append(result, bo)
		}

		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("close %q: %w", path, err)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no resources found in supplied manifests")
	}

	return result, nil
}

func validateObject(obj *unstructured.Unstructured) error {
	if obj.GetAPIVersion() == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if obj.GetKind() == "" {
		return fmt.Errorf("kind is required")
	}
	if obj.GetName() == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if _, err := schema.ParseGroupVersion(obj.GetAPIVersion()); err != nil {
		return fmt.Errorf("invalid apiVersion: %w", err)
	}
	return nil
}

func (r *reconciler) identity(
	buildObj *BuildObject,
) (ResourceID, error) {
	obj := buildObj.Object
	gv, err := schema.ParseGroupVersion(obj.GetAPIVersion())
	if err != nil {
		return ResourceID{}, err
	}

	mapping, err := r.mapper.RESTMapping(
		schema.GroupKind{
			Group: gv.Group,
			Kind:  obj.GetKind(),
		},
		gv.Version,
	)
	if err != nil {
		return ResourceID{}, fmt.Errorf("REST mapping: %w", err)
	}

	namespace := obj.GetNamespace()
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if buildObj.Metadata.Namespace != "" {
			namespace = buildObj.Metadata.Namespace
		} else if namespace == "" {
			namespace = "default"
		}
	} else if namespace != "" {
		return ResourceID{}, fmt.Errorf(
			"cluster-scoped resource %s must not specify namespace",
			obj.GetName(),
		)
	}

	return ResourceID{
		Group:     mapping.Resource.Group,
		Version:   mapping.Resource.Version,
		Resource:  mapping.Resource.Resource,
		Namespace: namespace,
		Name:      obj.GetName(),
	}, nil
}

func (r *reconciler) apply(
	ctx context.Context,
	obj *unstructured.Unstructured,
	id ResourceID,
) (Action, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if value := annotations[managedAnnotation]; value != "" &&
		value != fieldManager {
		return Action{}, fmt.Errorf(
			"reserved ownership annotation has unexpected value %q",
			value,
		)
	}

	annotations[managedAnnotation] = fieldManager
	obj.SetAnnotations(annotations)

	gv, err := schema.ParseGroupVersion(obj.GetAPIVersion())
	if err != nil {
		return Action{}, err
	}

	mapping, err := r.mapper.RESTMapping(
		schema.GroupKind{
			Group: gv.Group,
			Kind:  obj.GetKind(),
		},
		gv.Version,
	)
	if err != nil {
		return Action{}, err
	}

	var resource dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		resource = r.dynamic.Resource(mapping.Resource).Namespace(id.Namespace)
	} else {
		resource = r.dynamic.Resource(mapping.Resource)
	}

	body, err := json.Marshal(obj.Object)
	if err != nil {
		return Action{}, fmt.Errorf("marshal manifest: %w", err)
	}

	options := metav1.PatchOptions{
		FieldManager: fieldManager,
	}
	if r.dryRun {
		options.DryRun = []string{metav1.DryRunAll}
	}

	_, err = resource.Patch(
		ctx,
		id.Name,
		types.ApplyPatchType,
		body,
		options,
	)
	if err != nil {
		return Action{}, err
	}

	return Action{Operation: "apply", Resource: id}, nil
}

func (r *reconciler) prune(
	ctx context.Context,
	id ResourceID,
) (*Action, error) {
	if id.Group == "" && id.Resource == "configmaps" &&
		id.Namespace == inventoryNamespace && id.Name == inventoryName {
		return nil, fmt.Errorf("refusing to delete inventory ConfigMap")
	}

	gvr := schema.GroupVersionResource{
		Group:    id.Group,
		Version:  id.Version,
		Resource: id.Resource,
	}

	var resource dynamic.ResourceInterface
	if id.Namespace != "" {
		resource = r.dynamic.Resource(gvr).Namespace(id.Namespace)
	} else {
		resource = r.dynamic.Resource(gvr)
	}

	obj, err := resource.Get(ctx, id.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if obj.GetAnnotations()[managedAnnotation] != fieldManager {
		return nil, fmt.Errorf(
			"ownership check failed; refusing to delete %s",
			id,
		)
	}

	options := metav1.DeleteOptions{}
	if r.dryRun {
		options.DryRun = []string{metav1.DryRunAll}
	}

	err = resource.Delete(ctx, id.Name, options)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &Action{Operation: "delete", Resource: id}, nil
}

func (r *reconciler) loadInventory(
	ctx context.Context,
) ([]ResourceID, error) {
	cm, err := r.clientset.CoreV1().
		ConfigMaps(inventoryNamespace).
		Get(ctx, inventoryName, metav1.GetOptions{})

	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	raw := cm.Data[inventoryDataKey]
	if raw == "" {
		return nil, fmt.Errorf("inventory ConfigMap has no %q data",
			inventoryDataKey)
	}

	var stored storedInventory
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, fmt.Errorf("decode inventory: %w", err)
	}
	if stored.Version != inventoryVersion {
		return nil, fmt.Errorf(
			"unsupported inventory version %d", stored.Version,
		)
	}

	return stored.Resources, nil
}

func (r *reconciler) saveInventory(
	ctx context.Context,
	ids []ResourceID,
) error {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i].Key() < ids[j].Key()
	})

	raw, err := json.MarshalIndent(storedInventory{
		Version:   inventoryVersion,
		Resources: ids,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode inventory: %w", err)
	}

	client := r.clientset.CoreV1().ConfigMaps(inventoryNamespace)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := client.Get(
			ctx, inventoryName, metav1.GetOptions{},
		)

		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      inventoryName,
					Namespace: inventoryNamespace,
				},
				Data: map[string]string{
					inventoryDataKey: string(raw),
				},
			}, metav1.CreateOptions{})

			if apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("inventory ConfigMap was created concurrently")
			}
			return err
		}
		if err != nil {
			return err
		}

		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[inventoryDataKey] = string(raw)

		_, err = client.Update(ctx, cm, metav1.UpdateOptions{})
		return err
	})
}
