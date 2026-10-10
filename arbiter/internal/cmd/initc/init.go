package initc

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/exadrift/go/statusline"
	"github.com/exadrift/tools/arbiter/internal/config"
	"github.com/exadrift/tools/arbiter/internal/crypto"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "init",
		Short: "initialize talos project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initialize()
		},
	}

	return base
}

func initialize() error {
	sl := statusline.NewStatusLine()
	defer sl.Stop()

	sl.Emit("initializing arbiter project")

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	var (
		projectName string
		ipAddress   string
		configDir   string
	)
	isCorrect := false
	for !isCorrect {
		if !sl.Confirm(true, "project will be initialized at %s", dir) {
			sl.Emit("please re-run arbiter from desired location")
			return nil
		}

		if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
			ok := sl.Confirm(false, "you're not currently in a git repository (root dir), that ok?")
			if !ok {
				return nil
			}
		}

		_, dirEnd := filepath.Split(dir)
		projectName = strings.ToLower(dirEnd)
		if !sl.Confirm(true, "project name \"%s\"", projectName) {
			projectName = ""
			for len(projectName) == 0 {
				projectName = sl.Prompt(false, "name of project:")
				projectName = strings.Trim(projectName, " ")

				if len(projectName) == 0 {
					sl.Emit("❌ invalid project name")
				}

				configDir, err = config.GetConfigDir(projectName)
				if err != nil {
					return err
				}

				if _, err = os.Stat(configDir); err == nil {
					sl.Emit("❌ a project named \"%s\" already exists in the configuration location", projectName)
					projectName = ""
				}
			}
		}

		ipAddress := ""
		for len(ipAddress) == 0 {
			ipAddress = sl.Prompt(false, "ip address of the talos instance:")
			ipAddress = strings.Trim(ipAddress, " ")
			if _, err := netip.ParseAddr(ipAddress); err != nil {
				sl.Emit("❌ invalid ip address...")
			}
		}

		sl.Emit("\nproject location: %s\nproject name: %s\nip address: %s\n", dir, projectName, ipAddress)
		isCorrect = sl.Confirm(true, "is this information correct?")
	}

	sl.Status("initializing project...")

	// create the project config/key location
	if err = os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("unable to create config dir at %s: %w", configDir, err)
	}

	// create key material
	cr, err := crypto.New()
	if err != nil {
		return err
	}

	cfg, err := config.New(projectName, dir, ipAddress, cr)
	if err != nil {
		return err
	}

	if err = cfg.Save(); err != nil {
		return err
	}

	sl.Emit("✅ project created")

	return nil
}
