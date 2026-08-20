package branch

import (
	"fmt"
	"strings"

	"github.com/exadrift/tools/repo/internal/run"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "branch",
		Short: "git branch commands",
	}

	create := &cobra.Command{
		Use:   "create BRANCH",
		Short: "creates a remote branch and sets up tracking configs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createBranch(args[0])
		},
	}

	track := &cobra.Command{
		Use:   "track BRANCH",
		Short: "tracks a remote branch locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return trackBranch(args[0])
		},
	}

	prompt := &cobra.Command{
		Use:   "prompt",
		Short: "renders a shell prompt embed for the git branch",
		Run: func(cmd *cobra.Command, args []string) {
			branchPrompt()
		},
	}

	base.AddCommand(create)
	base.AddCommand(track)
	base.AddCommand(prompt)

	return base
}

func createBranch(name string) error {
	if err := run.Run("git", "checkout", "-b", name); err != nil {
		return err
	}

	if err := run.Run("git", "push", "origin", fmt.Sprintf("%s:refs/heads/%s", name, name)); err != nil {
		return err
	}

	if err := run.Run("git", "config", fmt.Sprintf("branch.%s.remote", name), "origin"); err != nil {
		return err
	}

	if err := run.Run("git", "config", fmt.Sprintf("branch.%s.merge", name), fmt.Sprintf("refs/heads/%s", name)); err != nil {
		return err
	}

	if err := run.Run("git", "config", fmt.Sprintf("branch.%s.rebase", name), "false"); err != nil {
		return err
	}

	return nil
}

func trackBranch(name string) error {
	if err := run.Run("git", "checkout", "-t", fmt.Sprintf("origin/%s", name)); err != nil {
		return err
	}

	if err := run.Run("git", "config", fmt.Sprintf("branch.%s.rebase", name), "false"); err != nil {
		return err
	}

	return nil
}

func branchPrompt() {
	branch, err := run.RunC("git", "branch", "--show-current")
	if err != nil {
		fmt.Println()
		return
	}

	branch = strings.Trim(branch, "\n")

	var branchChangeString string
	if err = run.Run("git", "diff", "--quiet"); err != nil {
		branchChangeString = "*"
	}

	fmt.Printf(" [%s%s]\n", branch, branchChangeString)
}
