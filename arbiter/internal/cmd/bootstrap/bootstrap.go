package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/exadrift/tools/arbiter/templates"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	"github.com/spf13/cobra"
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

	sl := statusline.NewStatusLine()
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

	var regenerate bool = true
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
		if err == nil {
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

	if err = authClient.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap failed with an error: %w", err)
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
	for {
		progress, err := stream.Recv()

		switch {
		case errors.Is(err, io.EOF):
			// The server completed the health-check stream normally.
			sl.Emit("✅ cluster is healthy - please securely back up cluster configuration (contains private keymaterial) located at ~/.arbiter/%s/config.yaml", name)
			return nil

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
}
