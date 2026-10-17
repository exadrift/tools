package bootstrap

import (
	"fmt"
	"log"
	"time"

	abconfig "github.com/exadrift/tools/arbiter/internal/config"
	"github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/constants"
)

func generateSecrets(cfg *abconfig.Config, versionContract *config.VersionContract) error {
	secretsBundle, err := secrets.NewBundle(
		secrets.NewFixedClock(time.Now()),
		versionContract,
	)
	if err != nil {
		log.Fatalf("generate secrets: %v", err)
	}

	return cfg.SaveMachineSecrets(secretsBundle)
}

func generateMachineConfigs(cfg *abconfig.Config, versionContract *config.VersionContract) error {
	if err := generateSecrets(cfg, versionContract); err != nil {
		return err
	}

	secretsBundle, err := cfg.LoadMachineSecrets()
	if err != nil {
		return err
	}

	kubeEndpoint := fmt.Sprintf("https://%s:6443", cfg.IpAddress)
	input, err := generate.NewInput(
		cfg.Name,
		kubeEndpoint,
		constants.DefaultKubernetesVersion,
		generate.WithVersionContract(versionContract),
		generate.WithSecretsBundle(secretsBundle),
		generate.WithEndpointList([]string{
			cfg.IpAddress,
		}),
		generate.WithSkipUnattendedInstallConfig(true),
	)
	if err != nil {
		return fmt.Errorf("unable to generate new input configs: %w", err)
	}

	controlPlaneConfig, err := input.Config(machine.TypeControlPlane)
	if err != nil {
		return fmt.Errorf("unable to generate control plane config: %w", err)
	}

	talosConfig, err := input.Talosconfig()
	if err != nil {
		return fmt.Errorf("unable to generate talosconfig: %w", err)
	}

	if err = cfg.SaveControlPlaneConfig(controlPlaneConfig); err != nil {
		return err
	}

	return cfg.SaveTalosConfig(talosConfig)
}
