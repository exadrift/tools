package bootstrap

import (
	"context"
	"fmt"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/shared"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "bootstrap [project-name]",
		Short: "bootstrap talos on target system",
		Long:  "boostrap talos either from the current project directory, or by passing the project name as the first command argument",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				err  error
				name string
			)
			if len(args) == 1 {
				name = args[0]
			} else {
				name, err = shared.GetNameFromRef()
				if err != nil {
					return err
				}
			}

			cmd.SilenceErrors = true
			return bootstrapCmd(name)
		},
	}

	return base
}

func bootstrapCmd(name string) error {
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}

	sl := statusline.NewStatusLine()
	defer sl.Stop()

	sl.Status("probing %s for maintenance mode", cfg.IpAddress)
	client, err := talossdk.NewClient(cfg, true)
	if err != nil {
		return fmt.Errorf("talos client creation failed: %w", err)
	}

	if err = client.Probe(context.Background()); err != nil {
		sl.Emit("❌ probe failed, verify node is in maintenance mode")
		return nil
	}
	sl.Emit("✅ node is in maintenance mode")

	return nil
}
