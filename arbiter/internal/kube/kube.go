package kube

import (
	"fmt"
	"os"
	"path/filepath"

	taloskubeconfig "github.com/siderolabs/talos/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func MergeKubeconfig(kubeConfig []byte, kubeconfigPath string, contextName string) error {
	// Parse and validate the kubeconfig returned by Talos.
	incoming, err := taloskubeconfig.LoadAndValidate(kubeConfig)
	if err != nil {
		return fmt.Errorf("validate Talos kubeconfig: %w", err)
	}

	// Give the new context a predictable name.
	if contextName != "" {
		for name, ctx := range incoming.Contexts {
			delete(incoming.Contexts, name)
			ctxName := contextName
			incoming.Contexts[ctxName] = ctx
			incoming.CurrentContext = ctxName
			break
		}
	}

	// Load the existing config, or start with an empty config.
	existing := api.NewConfig()
	if _, err := os.Stat(kubeconfigPath); err == nil {
		existing, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("load existing kubeconfig: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat kubeconfig: %w", err)
	}

	// Merge clusters, users, and contexts.
	for name, cluster := range incoming.Clusters {
		if _, exists := existing.Clusters[name]; exists {
			continue
		}
		existing.Clusters[name] = cluster
	}
	for name, user := range incoming.AuthInfos {
		if _, exists := existing.AuthInfos[name]; exists {
			continue
		}
		existing.AuthInfos[name] = user
	}
	for name, kubeContext := range incoming.Contexts {
		if _, exists := existing.Contexts[name]; exists {
			continue
		}
		existing.Contexts[name] = kubeContext
	}

	// Preserve the existing current context.
	if err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0700); err != nil {
		return fmt.Errorf("create kubeconfig directory: %w", err)
	}
	if err := clientcmd.WriteToFile(*existing, kubeconfigPath); err != nil {
		return fmt.Errorf("write merged kubeconfig: %w", err)
	}

	return os.Chmod(kubeconfigPath, 0600)
}
