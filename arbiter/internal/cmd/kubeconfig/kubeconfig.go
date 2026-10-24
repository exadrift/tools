package kubeconfigcmd

import (
	"context"
	"os"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/kubeconfig"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "kube-config [project-name]",
		Short: "generate kubernetes configuration",
		Long:  "generate kubernetes configuration",
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
			return kubeConfigCmd(name)
		},
	}

	return base
}

func kubeConfigCmd(name string) error {
	ctx := context.Background()
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}
	sl := statusline.NewStatusLine()
	defer sl.Stop()

	machineConfig, err := cfg.LoadMachineConfig()
	if err != nil {
		return err
	}

	client, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return err
	}

	if err = kubeconfig.DoKubeConfig(ctx, client, cfg, sl); err != nil {
		return err
	}

	return nil
}
