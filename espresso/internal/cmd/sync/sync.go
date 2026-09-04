package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/pshell"
	"github.com/exadrift/tools/espresso/internal/log"
	"github.com/exadrift/tools/espresso/internal/manifest"
	"github.com/spf13/cobra"
)

var (
	status  bool
	verbose bool
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "sync MANIFEST",
		Short: "synchronize package manifest to host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			var err error
			filename := args[0]
			if !filepath.IsAbs(filename) {
				filename, err = filepath.Abs(filename)
				if err != nil {
					return err
				}
			}

			if status {
				return Status(filename, verbose)
			}
			return Sync(filename, verbose)
		},
	}

	base.Flags().BoolVarP(&status, "status", "s", false, "establishes the current sync status without syncing")
	base.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose error reporting")

	return base
}

func resolveSyncScript(filename string) (string, error) {
	var err error
	absPath := filename
	if !filepath.IsAbs(filename) {
		// turn relative paths into absolute
		baseDir := filepath.Dir(filename)
		absPath, err = filepath.Abs(filepath.Join(baseDir, filename))
		if err != nil {
			return "", err
		}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", fmt.Errorf("script is not a regular file")
	}

	return absPath, nil
}

func Status(filename string, verbose bool) error {
	l := log.New()
	defer l.Stop()
	l.Status(style.T("validating manifest..."))

	ps := pshell.New(pshell.WithPasswordCallback(func() (string, error) {
		return l.Prompt(style.T("[sudo: authenticate] Password:")), nil
	}))

	mf, err := manifest.FromFile(filename)
	if err != nil {
		l.EmitCondition(log.ConditionFail, style.T("manifest failed validation"))
		return err
	} else {
		l.EmitCondition(log.ConditionPass, style.T("manifest passed validation"))
	}

	for _, pkg := range mf.Packages {
		l.Status(style.T("checking package ", style.Cyan.Fg(), pkg.Name))

		version, err := ps.ExecuteCommands(pkg.GetVersion)
		if err != nil {
			if verbose {
				l.EmitRaw(err.Error())
			}
			l.EmitCondition(log.ConditionFail, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " is not installed"))
			continue
		} else {
			if verbose {
				l.EmitRaw(version)
			}
		}

		if version != pkg.Version {
			l.EmitCondition(log.ConditionWarn, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " version ", style.Yellow.Fg(), version, style.StyleReset, " out of sync with ", style.Yellow.Fg(), pkg.Version))
		} else {
			l.EmitCondition(log.ConditionPass, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " is in sync"))
		}
	}

	return nil
}

func Sync(filename string, verbose bool) error {
	l := log.New()
	defer l.Stop()
	l.Status(style.T("validating manifest..."))

	ps := pshell.New(pshell.WithPasswordCallback(func() (string, error) {
		return l.Prompt(style.T("[sudo: authenticate] Password:")), nil
	}))

	mf, err := manifest.FromFile(filename)
	if err != nil {
		l.EmitCondition(log.ConditionFail, style.T("manifest failed validation"))
		return err
	}
	l.EmitCondition(log.ConditionPass, style.T("manifest passed validation"))

	hasErrors := false
	for _, pkg := range mf.Packages {
		l.Status(style.T("checking package ", style.Cyan.Fg(), pkg.Name))

		var stdout string
		if pkg.SyncScript != "" {
			pkg.SyncScript, err = resolveSyncScript(pkg.SyncScript)
			if err != nil {
				hasErrors = true
				l.EmitCondition(log.ConditionFail, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, fmt.Sprintf(" error with sync script path %s", err.Error())))
				continue
			}
		}

		version, err := ps.ExecuteCommands(pkg.GetVersion)
		if err != nil {
			if verbose {
				l.EmitRaw(err.Error())
			}
			l.Emit(style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " is not installed"))
		} else {
			if verbose {
				l.EmitRaw(stdout)
			}
			if version == pkg.Version {
				l.EmitCondition(log.ConditionPass, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " version ", style.Yellow.Fg(), version, style.StyleReset, " is in sync"))
				continue
			} else {
				l.Emit(style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " version ", style.Yellow.Fg(), version, style.StyleReset, " out of sync with ", style.Yellow.Fg(), pkg.Version))
			}
		}

		l.Status(style.T("installing package ", style.Cyan.Fg(), pkg.Name))
		if pkg.SyncScript != "" {
			stdout, err = ps.Execute(pkg.SyncScript, pshell.WithEnvVars(map[string]string{"VERSION": pkg.Version}))
		} else {
			stdout, err = ps.ExecuteCommands(pkg.Sync, pshell.WithEnvVars(map[string]string{"VERSION": pkg.Version}))
		}
		if err != nil {
			hasErrors = true
			l.EmitCondition(log.ConditionFail, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " installation failed"))
			if verbose {
				l.EmitRaw(err.Error())
			}
			continue
		} else {
			if verbose {
				l.EmitRaw(stdout)
			}
		}
		l.EmitCondition(log.ConditionPass, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " installed"))

		l.Status(style.T("verifying package ", style.Cyan.Fg(), pkg.Name))
		version, err = ps.ExecuteCommands(pkg.GetVersion)
		if err != nil {
			hasErrors = true
			l.EmitCondition(log.ConditionFail, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " could not be verified"))
			if verbose {
				l.EmitRaw(err.Error())
			}
			continue
		} else {
			if verbose {
				l.EmitRaw(stdout)
			}
		}

		if version != pkg.Version {
			hasErrors = true
			l.EmitCondition(log.ConditionFail, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " version ", style.Yellow.Fg(), version, style.StyleReset, " out of sync with ", style.Yellow.Fg(), pkg.Version))
			continue
		}

		l.EmitCondition(log.ConditionPass, style.T("package ", style.Cyan.Fg(), pkg.Name, style.StyleReset, " verified"))
	}

	if hasErrors {
		if verbose {
			return fmt.Errorf("one or more errors occurred during sync")
		}
		l.EmitCondition(log.ConditionFail, style.T("one or more error occurred during sync"))
		return fmt.Errorf("one or more errors occurred during sync, use the --verbose option for full details")
	}
	l.EmitCondition(log.ConditionPass, style.T("sync successful"))
	return nil
}
