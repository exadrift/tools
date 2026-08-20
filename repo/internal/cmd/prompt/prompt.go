package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/exadrift/tools/repo/internal/run"
	"github.com/spf13/cobra"
)

var (
	Username = run.ShellRgb(52, 103, 235, `\u`)
	At       = run.ShellRgb(154, 203, 255, "@")
	Hostname = run.ShellRgb(52, 103, 235, `\h`)
	Path     = run.ShellRgb(165, 32, 227, `\w`)
	Branch   = run.ShellRgb(250, 137, 0, "${BRANCH_CMD}")
	Embed    = "PROMPT_COMMAND=$(repo prompt render)"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "prompt",
		Short: "shell prompt commands",
	}

	render := &cobra.Command{
		Use:   "render",
		Short: "renders a shell prompt string",
		Run: func(cmd *cobra.Command, args []string) {
			renderPrompt()
		},
	}

	inject := &cobra.Command{
		Use:   "inject",
		Short: "injects the prompt into the shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			return injectPrompt()
		},
	}

	base.AddCommand(render)
	base.AddCommand(inject)

	return base
}

func renderPrompt() {
	promptString := fmt.Sprintf("BRANCH_CMD=$(repo branch prompt); PS1='%s%s%s:%s%s $ '", Username, At, Hostname, Path, Branch)
	fmt.Println(promptString)
}

func injectPrompt() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	rcPath := filepath.Join(homeDir, ".bashrc")
	fBytes, err := os.ReadFile(rcPath)
	if err != nil {
		return err
	}

	rcData := string(fBytes)
	if !strings.Contains(rcData, Embed) {
		rcData = fmt.Sprintf("%s\n%s\n", rcData, Embed)
		if err = os.WriteFile(rcPath, []byte(rcData), 0644); err != nil {
			return err
		}
		fmt.Printf("prompt updated, please run\n\nsource %s\n\nor restart the shell\n", rcPath)
	} else {
		fmt.Printf("prompt was already found in %s\n", rcPath)
	}

	return nil
}
