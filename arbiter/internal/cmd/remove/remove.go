package remove

import (
	"os"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/utils"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "remove [project-name]",
		Short: "remove arbiter project",
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
			return removeCmd(name)
		},
	}

	return base
}

func removeCmd(name string) error {
	_, err := config.Load(name)
	if err != nil {
		return err
	}

	sl := utils.NewStatusLine()
	defer sl.Stop()

	confirm := sl.Confirm(false, "are you sure you want to remove the project: %s?", name)
	if !confirm {
		sl.Emit("aborted...")
		return nil
	}

	configDir, err := config.GetConfigDir(name)
	if err != nil {
		return err
	}

	if err = os.RemoveAll(configDir); err != nil {
		return err
	}

	sl.Emit("✅ project removed")

	return nil
}
