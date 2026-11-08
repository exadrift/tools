package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/kubeconfig"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/exadrift/tools/arbiter/internal/utils"
	"github.com/exadrift/tools/arbiter/templates"
	"github.com/ghodss/yaml"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	HelmRepoUrl = "https://argoproj.github.io/argo-helm"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "bootstrap [project-name]",
		Short: "bootstrap talos on target system",
		Long:  "boostrap talos either from the current project directory, or by passing the project name as the first command argument",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				return cerr
			}
			var (
				err  error
				name string
			)
			if len(args) == 1 {
				name = args[0]
			} else {
				name, err = config.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cmd.SilenceUsage = true
			return bootstrapCmd(name)
		},
	}

	return base
}

func bootstrapCmd(name string) error {
	ctx := context.Background()
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}

	sl := utils.NewStatusLine()
	defer sl.Stop()

	sl.Emit("bootstrapping \"%s\"", cfg.IpAddress)
	sl.Status("probing for maintenance mode")
	unauthClient, err := talossdk.NewMaintenanceClient(cfg)
	if err != nil {
		return fmt.Errorf("talos client creation failed: %w", err)
	}
	defer func() {
		_ = unauthClient.Close()
	}()

	versionContract, err := unauthClient.GetVersion(ctx)
	if err != nil {
		sl.Emit("❌ probe failed, verify node is in maintenance mode")
		sl.Emit("error message: %s", err.Error())
		return nil
	}
	sl.Emit("✅ node is in maintenance mode, running %s", versionContract.String())

	var regenerate bool
	sl.Status("checking machine configuration")
	machineConfig, err := cfg.LoadMachineConfig()
	if err == nil {
		regenerate = sl.Confirm(false, "machine configs have already been generated, generate new ones?")
	}

	if regenerate {
		sl.Status("generating new machine configuration")

		// identify disks
		resp, err := unauthClient.Disks(ctx)
		if err != nil {
			return fmt.Errorf("unable to obtain machine disks: %w", err)
		}

		var choices []string
		var dSelect []*storage.Disk
		i := 0
		for _, m := range resp.Messages {
			for _, disk := range m.Disks {
				choices = append(choices, fmt.Sprintf("%d - %s %s ( %s )", i+1, disk.Model, disk.Name, humanize.Bytes(disk.Size)))
				dSelect = append(dSelect, disk)
				i++
			}
		}

		var selectedDiskIndex int
		sl.Emit("available disks:\n%s\n", strings.Join(choices, "\n"))
		for {
			installDiskIndexStr := sl.Prompt(false, "which disk should talos be installed on?")
			installDiskNum, err := strconv.ParseInt(installDiskIndexStr, 10, 64)
			if err == nil && installDiskNum > 0 && installDiskNum <= int64(len(dSelect)) {
				selectedDiskIndex = int(installDiskNum - 1)
				break
			}
		}

		disk := dSelect[selectedDiskIndex]
		installConfigBytes, err := templates.RenderDiskTemplate(disk.Serial)
		if err != nil {
			return fmt.Errorf("unable to render install disk template: %w", err)
		}

		if err = cfg.StoreData(config.InstallDiskFilename, installConfigBytes); err != nil {
			return err
		}

		if err = generateMachineConfigs(cfg, versionContract); err != nil {
			return err
		}

		machineConfig, err = cfg.LoadMachineConfig()
		if err != nil {
			return fmt.Errorf("unable to generate machine configs: %w", err)
		}
	}

	controlPlaneBytes, err := machineConfig.RenderControlPlane()
	if err != nil {
		return err
	}

	sl.Status("applying machine configuration to host")

	if err = unauthClient.ApplyConfiguration(ctx, controlPlaneBytes, false); err != nil {
		return err
	}
	_ = unauthClient.Close()

	authClient, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return fmt.Errorf("unable to create authenticated talos client: %w", err)
	}
	defer func() {
		_ = authClient.Close()
	}()

	start := time.Now()
	healthy := false
	for time.Since(start) < time.Minute*15 {
		sl.Status("waiting for node to become healthy...")
		time.Sleep(3 * time.Second)
		if _, err := authClient.GetVersion(ctx); err == nil {
			sl.Emit("✅ node became healthy")
			healthy = true
			break
		}
	}
	if !healthy {
		sl.Emit("❌ timeout waiting for node to become healthy")
		return nil
	}

	sl.Status("waiting for node to apply configs before bootstrap")
	for {
		if err = authClient.Bootstrap(ctx); err != nil {
			if status.Code(err) == codes.FailedPrecondition {
				time.Sleep(time.Second * 5)
			} else {
				return fmt.Errorf("bootstrap failed with an error: %w", err)
			}
		} else {
			sl.Emit("✅ bootstrap initiated")
			break
		}
	}

	healthCtx, cancelHealth := context.WithTimeout(
		ctx,
		15*time.Minute,
	)
	defer cancelHealth()

	sl.Status("performing cluster health check")
	sl.HoldTimer()
	defer sl.ReleaseTimer()
	stream, err := authClient.HealthCheck(healthCtx, cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize health check client: %w", err)
	}
outer:
	for {
		progress, err := stream.Recv()

		switch {
		case errors.Is(err, io.EOF):
			// The server completed the health-check stream normally.
			sl.Emit("✅ cluster is ready for use")
			break outer

		case err != nil:
			// Includes RPC failures and context cancellation/deadline.
			sl.Emit("❌ cluster health check failed: %s", err.Error())
			return nil

		default:
			if progress != nil && progress.GetMessage() != "" {
				sl.Status("performing cluster health check: %s", progress.GetMessage())
			}
		}
	}
	sl.ReleaseTimer()

	sl.Status("preparing kubernetes configuration")
	if err = kubeconfig.DoKubeConfig(context.Background(), authClient, cfg, sl); err != nil {
		return err
	}

	gitUrl, err := cfg.GetGitUrl()
	if err != nil {
		return err
	}

	bootstrapBytes, err := templates.RenderBootstrapApplicationTemplate(templates.BootstrapApplicationConfig{
		Name: strings.ToLower(cfg.Name) + "-applications",
		Repo: gitUrl,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(cfg.ProjectDir, "bootstrap.yaml"), bootstrapBytes, 0644); err != nil {
		return err
	}

	sl.Status("installing argocd")
	version, err := installArgoCd(ctx, sl, cfg)
	if err != nil {
		return fmt.Errorf("error installing argocd: %w", err)
	}

	argocdApplicationBytes, err := templates.RenderArgoCdApplicationTemplate(templates.ArgoCdApplicationConfig{
		HelmRepo: HelmRepoUrl,
		Repo:     gitUrl,
		Version:  version,
	})
	if err != nil {
		return err
	}

	if err = cfg.WriteApplicationTemplate("infra/argocd", "application.yaml", argocdApplicationBytes); err != nil {
		return err
	}

	if err = cfg.WriteApplicationTemplate("infra/argocd", "values.yaml", nil); err != nil {
		return err
	}

	sl.Emit("to configure additional resources such as disk, etc., please run \"arbiter post-configure\"")
	return nil
}

func getLatestVersion(settings *cli.EnvSettings, repoUrl string) (string, error) {
	indexUrl, err := url.JoinPath(repoUrl, "/index.yaml")
	if err != nil {
		return "", err
	}

	getters := getter.All(settings)

	httpGetter, err := getters.ByScheme("https")
	if err != nil {
		return "", fmt.Errorf("Failed to find getter for https: %w", err)
	}

	res, err := httpGetter.Get(indexUrl)
	if err != nil {
		return "", fmt.Errorf("Failed to download index: %w", err)
	}

	indexFile := repo.NewIndexFile()
	if err := yaml.Unmarshal(res.Bytes(), indexFile); err != nil {
		return "", fmt.Errorf("Failed to unmarshal index YAML: %w", err)
	}
	indexFile.SortEntries()

	versions, exists := indexFile.Entries["argo-cd"]
	if !exists || len(versions) == 0 {
		return "", fmt.Errorf("argo-cd chart not found in repository index")
	}

	return versions[0].Version, nil
}

func installArgoCd(ctx context.Context, sl *statusline.StatusLine, cfg *config.Config) (string, error) {
	machineConfig, err := cfg.LoadMachineConfig()
	if err != nil {
		return "", err
	}

	client, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return "", err
	}

	kubeClientConfigBytes, err := client.KubeConfig(ctx)
	if err != nil {
		return "", err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeClientConfigBytes)
	if err != nil {
		return "", err
	}

	// Convert to Kubernetes REST config.
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return "", err
	}

	// Configure Helm to use our in-memory REST config.
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.WrapConfigFn = func(_ *rest.Config) *rest.Config {
		return restConfig
	}

	// Initialize Helm's action configuration.
	actionConfig := new(action.Configuration)

	if err = actionConfig.Init(
		kubeConfig,
		"argocd",
		os.Getenv("HELM_DRIVER"),
		sl.Emit,
	); err != nil {
		return "", err
	}

	settings := cli.New()
	version, err := getLatestVersion(settings, HelmRepoUrl)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve latest manifest version: %w", err)
	}
	sl.Emit("installing version %s of argo cd", version)

	// Create the Helm install action.
	hClient := action.NewInstall(actionConfig)

	hClient.ReleaseName = "argocd"
	hClient.Namespace = "argocd"
	hClient.CreateNamespace = true
	hClient.Wait = true
	hClient.Timeout = 5 * 60 * 1000000000 // 5 minutes

	// Configure the chart repository.
	hClient.ChartPathOptions.RepoURL = HelmRepoUrl
	hClient.ChartPathOptions.Version = version

	// Download the chart.
	chartPath, err := hClient.ChartPathOptions.LocateChart(
		"argo-cd",
		settings,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Load the chart.
	chart, err := loader.Load(chartPath)
	if err != nil {
		log.Fatal(err)
	}

	// Supply custom Helm values.
	values := map[string]interface{}{
		"configs": map[string]interface{}{
			"params": map[string]interface{}{
				"server.insecure": true,
			},
		},
	}

	// Install the chart.
	release, err := hClient.RunWithContext(
		context.Background(),
		chart,
		values,
	)
	if err != nil {
		log.Fatal(err)
	}

	sl.Emit(
		"✅ successfully installed release %s in namespace %s\n",
		release.Name,
		release.Namespace,
	)

	return version, nil
}
