package export

import (
	"fmt"
	"os"
	"path/filepath"

	intconfig "github.com/exadrift/tools/arbiter/internal/config"
	"github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/config/generate/secrets"
	"github.com/siderolabs/talos/pkg/machinery/role"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "export [project-name]",
		Short: "exports a config for use with talosctl",
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
				name, err = intconfig.GetNameRef(cwd)
				if err != nil {
					return err
				}
			}

			cmd.SilenceUsage = true
			return applyCmd(name)
		},
	}

	return base
}

func applyCmd(name string) error {
	cfg, err := intconfig.Load(name)
	if err != nil {
		return err
	}

	secretsBundle, err := cfg.LoadMachineSecrets()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	talosDir := filepath.Join(homeDir, ".talos")

	if err = os.MkdirAll(talosDir, 0700); err != nil {
		return err
	}

	talosConfigFilename := filepath.Join(talosDir, "config")
	if err = writeTalosconfig(
		secretsBundle,
		cfg.Name,
		cfg.IpAddress,
		talosConfigFilename,
	); err != nil {
		return err
	}

	fmt.Printf("config exported to %s\n", talosConfigFilename)
	return nil
}

func writeTalosconfig(
	secretsBundle *secrets.Bundle,
	clusterName string,
	ipAddress string,
	outPath string,
) error {
	clientCert, err :=
		secretsBundle.GenerateTalosAPIClientCertificate(
			role.MakeSet(role.Admin),
		)
	if err != nil {
		return err
	}

	endpoints := []string{ipAddress}

	talosConfig := config.NewConfig(
		clusterName,
		endpoints,
		secretsBundle.Certs.OS.Crt,
		clientCert,
	)

	talosConfig.Contexts[talosConfig.Context].Nodes =
		[]string{ipAddress}

	yamlBytes, err := talosConfig.Bytes()
	if err != nil {
		return err
	}

	return os.WriteFile(outPath, yamlBytes, 0o600)
}
