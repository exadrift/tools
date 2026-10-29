package postconfigure

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/exadrift/tools/arbiter/internal/argocd"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/exadrift/tools/arbiter/internal/utils"
	"github.com/exadrift/tools/arbiter/templates"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "post-configure [project-name]",
		Short: "configures additional initial resources on a running talos system",
		Long:  "configures additional initial resources on a running talos system",
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
			return postConfigCmd(name)
		},
	}

	return base
}

func postConfigCmd(name string) error {
	ctx := context.Background()
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}

	talosPatchIndex, err := cfg.GetCurrentPatchIndex(config.PatchTypeTalos)
	if err != nil {
		return fmt.Errorf("unable to analyze current patch directory: %w", err)
	}

	kubePatchIndex, err := cfg.GetCurrentPatchIndex(config.PatchTypeKubernetes)
	if err != nil {
		return fmt.Errorf("unable to analyze current patch directory: %w", err)
	}

	sl := utils.NewStatusLine()
	defer sl.Stop()

	if talosPatchIndex > 1 || kubePatchIndex > 1 {
		wipePatches := sl.Confirm(false, "you already have some kube/talos patches, should i erase them (if not new ones will be added)?")
		if wipePatches {
			err = os.RemoveAll(cfg.GetPatchDir(config.PatchTypeTalos))
			if err != nil {
				return err
			}
			err = os.RemoveAll(cfg.GetPatchDir(config.PatchTypeKubernetes))
			if err != nil {
				return err
			}
			talosPatchIndex = 1
			kubePatchIndex = 1
		}
	}

	addStorage := sl.Confirm(true, "would you like to add any storage volumes to kubernetes?")

	machineConfig, err := cfg.LoadMachineConfig()
	if err != nil {
		return err
	}
	client, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return err
	}

	for addStorage {
		resp, err := client.Disks(ctx)
		if err != nil {
			return fmt.Errorf("unable to retrieve disks: %w", err)
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
			installDiskIndexStr := sl.Prompt(false, "which disk will be used for the storage volume?")
			installDiskNum, err := strconv.ParseInt(installDiskIndexStr, 10, 64)
			if err == nil && installDiskNum > 0 && installDiskNum <= int64(len(dSelect)) {
				selectedDiskIndex = int(installDiskNum - 1)
				break
			}
		}

		disk := dSelect[selectedDiskIndex]
		absMaxSize := disk.Size / (1024 * 1024 * 1024)
		minSize := 1
		maxSize := 1

		for {
			spaceStr := sl.Prompt(false, "how much space in GB should be allocated to the volume (max %d) <default to max>?", absMaxSize)
			if spaceStr == "" {
				minSize = int(absMaxSize)
				break
			}
			volSize, err := strconv.ParseInt(spaceStr, 10, 64)
			if err == nil && volSize > 0 && volSize < int64(absMaxSize) {
				minSize = int(volSize)
				break
			}
		}

		if minSize < int(absMaxSize) {
			for {
				spaceStr := sl.Prompt(false, "how much of the max should the volume be growable to (max %d) <default to max>?", absMaxSize)
				if spaceStr == "" {
					maxSize = int(absMaxSize)
					break
				}
				volSize, err := strconv.ParseInt(spaceStr, 10, 64)
				if err == nil && volSize > 0 && volSize < int64(absMaxSize) {
					maxSize = int(volSize)
					break
				}
			}
		} else {
			maxSize = minSize
		}

		var volName string
		for {
			volName = sl.Prompt(false, "what should this storage volume be called?")
			volName = strings.ToLower(strings.Trim(volName, " \t"))
			if volName != "" {
				break
			}
		}

		volData, err := templates.RenderStorageVolumeTemplate(templates.StorageVolumeConfig{
			Name:    volName,
			Serial:  disk.Serial,
			MinSize: minSize,
			MaxSize: maxSize,
		})
		if err != nil {
			return fmt.Errorf("error rendering storage volume template: %w", err)
		}

		if err = cfg.SavePatch(config.PatchTypeTalos, fmt.Sprintf("%04d-storage-volume-%s", talosPatchIndex, volName), volData, nil); err != nil {
			return err
		}
		talosPatchIndex++

		pvData, err := templates.RenderPersistentVolumeTemplate(templates.PersistentVolumeConfig{
			Name:     volName,
			Size:     minSize,
			Hostname: cfg.Name,
		})
		if err != nil {
			return fmt.Errorf("error rendering persistent volume template: %w", err)
		}

		if err = cfg.SavePatch(config.PatchTypeKubernetes, fmt.Sprintf("%04d-persistent-volume-%s", kubePatchIndex, volName), pvData, nil); err != nil {
			return err
		}
		kubePatchIndex++

		addStorage = sl.Confirm(false, "add additional storage?")
	}

	setupArgo := sl.Confirm(true, "would you like to setup argocd?")
	if setupArgo {
		manifest, err := argocd.GetArgoCdFullManifest()
		if err != nil {
			return err
		}

		patchMeta := &config.PatchMetadata{
			Namespace: "argocd",
		}
		if err = cfg.SavePatch(config.PatchTypeKubernetes, fmt.Sprintf("%04d-argocd", kubePatchIndex), manifest, patchMeta); err != nil {
			return err
		}
	}

	sl.Emit("✅ patches updated, run `arbiter apply` to apply changes")

	return nil
}
