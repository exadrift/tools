package apply

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/kube"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/exadrift/tools/arbiter/internal/utils"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "apply [project-name]",
		Short: "applies system configuration to a running talos instance",
		Long:  "applies system configuration to a running talos instance",
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
			return applyCmd(name)
		},
	}

	return base
}

func applyCmd(name string) error {
	ctx := context.Background()
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}

	sl := utils.NewStatusLine()
	defer sl.Stop()

	machineConfig, err := cfg.LoadMachineConfig()
	if err != nil {
		return err
	}

	hardwareConfig, err := cfg.LoadTalosPatchGroup()
	if err != nil {
		return err
	}

	client, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return err
	}

	patchFiles, err := cfg.GetOrderedPatchGroupFilenames(config.PatchTypeTalos)
	if err != nil {
		return err
	}
	for _, patchFile := range patchFiles {
		sl.Emit("preparing %s", patchFile)
	}

	if err = client.ApplyConfiguration(context.Background(), hardwareConfig, true); err != nil {
		sl.Emit("❌ dry run failed - %s", err.Error())
		return nil
	}

	sl.Emit("✅ dry run successful")

	if err = client.ApplyConfiguration(context.Background(), hardwareConfig, false); err != nil {
		sl.Emit("❌ failed to apply configuration - %s", err.Error())
		return nil
	}

	healthCtx, cancelHealth := context.WithTimeout(
		ctx,
		15*time.Minute,
	)
	defer cancelHealth()

	sl.Status("performing cluster health check")
	sl.HoldTimer()
	defer sl.ReleaseTimer()
	stream, err := client.HealthCheck(healthCtx, cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize health check client: %w", err)
	}
outer:
	for {
		progress, err := stream.Recv()

		switch {
		case errors.Is(err, io.EOF):
			// The server completed the health-check stream normally.
			sl.Emit("✅ cluster is healthy")
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

	manifestFiles, err := cfg.GetOrderedPatchGroupFilenames(config.PatchTypeKubernetes)
	if err != nil {
		return err
	}
	sl.Status("planning dry run...")
	if len(manifestFiles) > 0 {
		report, err := kube.Reconcile(
			ctx,
			sl,
			cfg.KubeConfigPath,
			cfg.KubeContextName,
			manifestFiles,
			true,
		)
		if err != nil {
			return err
		}

		for _, action := range report.Actions {
			sl.Emit("%s: %s", action.Operation, action.Resource)
		}

		sl.Status("reconciling resources...")
		_, err = kube.Reconcile(
			ctx,
			sl,
			cfg.KubeConfigPath,
			cfg.KubeContextName,
			manifestFiles,
			false,
		)
		if err != nil {
			return err
		}

		sl.Emit("✅ kubernetes resources reconcilled")
	}

	return nil
}
