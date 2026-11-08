package info

import (
	"context"
	"fmt"
	"os"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/talossdk"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "info [project-name]",
		Short: "gets info about the talos instance",
		Long:  "gets info about the talos instance",
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
			return infoCmd(name)
		},
	}

	return base
}

func infoCmd(name string) error {
	ctx := context.Background()
	cfg, err := config.Load(name)
	if err != nil {
		return err
	}

	machineConfig, err := cfg.LoadMachineConfig()
	if err != nil {
		return err
	}

	client, err := talossdk.NewAuthenticatedClient(machineConfig)
	if err != nil {
		return err
	}

	links, err := client.NetworkInterfaces(ctx)
	if err != nil {
		return err
	}

	addresses, err := client.NetworkAddresses(ctx)
	if err != nil {
		return err
	}

	for _, link := range links.Items {
		linkID := link.Metadata().ID()

		fmt.Printf("Link: %s\n", linkID)

		for _, address := range addresses.Items {
			addr := address.(*network.AddressStatus)

			if addr.TypedSpec().LinkName == string(linkID) {
				fmt.Printf("  Address: %s\n", addr.TypedSpec().Address)
			}
		}
	}

	return nil
}
