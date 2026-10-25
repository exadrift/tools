package list

import (
	"fmt"
	"os"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "list",
		Short: "list arbiter projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return listCmd()
		},
	}

	return base
}

func listCmd() error {
	dir, err := config.GetConfigDirBase()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println(entry.Name())
		}
	}

	return nil
}
