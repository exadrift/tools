package restore

import (
	"os"
	"strings"

	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/utils"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "restore configPath",
		Short: "restore talos config from config.yaml",
		Long:  "restore talos config from config.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return restoreCmd(args[0])
		},
	}

	return base
}

func restoreCmd(path string) error {
	cfg, err := config.LoadFromAltLocation(path, "")
	if err != nil {
		return err
	}

	var projectName string
	sl := utils.NewStatusLine()
	defer sl.Stop()

	keep := sl.Confirm(true, "current project name is \"%s\" are do you want to keep it?", cfg.Name)
	if !keep {
		for projectName == "" {
			projectName = sl.Prompt(false, "enter a new project name:")
			projectName = strings.Trim(projectName, " \t")
		}

		cfg.Name = projectName
	}

	for {
		_, err := cfg.LoadMachineConfig()
		if err == nil {
			sl.Emit("✅ project directory integrity matches config")
			break
		}

		sl.Emit("❌ the directory \"%s\" doesn't match the secret config provided the config file we're restoring", cfg.ProjectDir)
		cfg.ProjectDir = sl.Prompt(false, "please specify the companion project directory")
	}

	if err = cfg.SaveNameRef(); err != nil {
		return err
	}

	configDir := cfg.GetConfigDir()
	if _, err := os.Stat(configDir); err == nil {
		replace := sl.Confirm(false, "there's already config located at \"%s\", would you like to replace it with this one?", configDir)
		if !replace {
			sl.Emit("❌ configuration was not restored, aborting...")
			return nil
		}

		if err = os.RemoveAll(configDir); err != nil {
			return err
		}
	}

	if err = cfg.Save(); err != nil {
		return err
	}

	sl.Emit("✅ config successfully restored")

	return nil
}
