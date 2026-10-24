package kubeconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/kube"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func DoKubeConfig(ctx context.Context, client *talossdk.TalosClient, cfg *config.Config, sl *statusline.StatusLine) error {
	kubeConfig, err := client.KubeConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to provision kube config: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	if err = os.MkdirAll(kubeDir, 0700); err != nil {
		return err
	}

	mergeKube := sl.Confirm(true, "would you like to merge kube config for this cluster with your existing kube config?")
	if !mergeKube {
		kubeConfigFile := filepath.Join(kubeDir, cfg.Name)
		_ = os.Remove(kubeConfigFile)
		if err = kube.MergeKubeconfig(kubeConfig, kubeConfigFile, cfg.Name); err != nil {
			return err
		}
		sl.Emit("✅ kube config created at %s", kubeConfigFile)

		cfg.KubeConfigPath = kubeConfigFile
		cfg.KubeContextName = cfg.Name
		if err = cfg.Save(); err != nil {
			return err
		}

		return nil
	}

	kubeConfigFile := filepath.Join(kubeDir, "config")
	if err = kube.MergeKubeconfig(kubeConfig, kubeConfigFile, cfg.Name); err != nil {
		return err
	}

	cfg.KubeConfigPath = kubeConfigFile
	cfg.KubeContextName = cfg.Name
	if err = cfg.Save(); err != nil {
		return err
	}

	sl.Emit("✅ kube config updated with new context: %s", cfg.Name)

	return nil
}

func LoadKubeConfig(kubeConfigPath string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeConfigPath

	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{},
	)

	config, err := loader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	config.UserAgent = "github.com/exadrift/tools/arbiter"

	return config, nil
}
