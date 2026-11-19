package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/exadrift/tools/arbiter/internal/cmd/apply"
	"github.com/exadrift/tools/arbiter/internal/cmd/bootstrap"
	"github.com/exadrift/tools/arbiter/internal/cmd/export"
	"github.com/exadrift/tools/arbiter/internal/cmd/info"
	"github.com/exadrift/tools/arbiter/internal/cmd/initc"
	kubeconfigcmd "github.com/exadrift/tools/arbiter/internal/cmd/kubeconfig"
	"github.com/exadrift/tools/arbiter/internal/cmd/list"
	postconfigure "github.com/exadrift/tools/arbiter/internal/cmd/post-configure"
	"github.com/exadrift/tools/arbiter/internal/cmd/remove"
	"github.com/exadrift/tools/arbiter/internal/cmd/restore"
	"github.com/exadrift/tools/arbiter/internal/cmd/secrets"
	"github.com/spf13/cobra"
)

// Version holds the version of the arbiter tool
// This is set during build time using ldflags
var Version string

// main is the entry point for the arbiter command-line tool
// It sets up the command structure and executes the appropriate command based on user input
func main() {
	if slices.Contains(os.Args, "--version") {
		fmt.Println(Version)
		return
	}

	rootCmd := &cobra.Command{
		Use:   "arbiter",
		Short: "A guided, GitOps-oriented tool for Talos cluster management",
		Long:  "Arbiter is a tool designed to manage single-node Talos clusters with GitOps workflows, focusing on reproducibility, ease of management, and secure secret handling.",
	}

	rootCmd.AddCommand(initc.Command())
	rootCmd.AddCommand(bootstrap.Command())
	rootCmd.AddCommand(restore.Command())
	rootCmd.AddCommand(postconfigure.Command())
	rootCmd.AddCommand(apply.Command())
	rootCmd.AddCommand(kubeconfigcmd.Command())
	rootCmd.AddCommand(list.Command())
	rootCmd.AddCommand(remove.Command())
	rootCmd.AddCommand(secrets.Command())
	rootCmd.AddCommand(info.Command())
	rootCmd.AddCommand(export.Command())
	_ = rootCmd.Execute()
}
