package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/exadrift/tools/arbiter/internal/cmd/apply"
	"github.com/exadrift/tools/arbiter/internal/cmd/bootstrap"
	"github.com/exadrift/tools/arbiter/internal/cmd/initc"
	kubeconfigcmd "github.com/exadrift/tools/arbiter/internal/cmd/kubeconfig"
	"github.com/exadrift/tools/arbiter/internal/cmd/list"
	postconfigure "github.com/exadrift/tools/arbiter/internal/cmd/post-configure"
	"github.com/exadrift/tools/arbiter/internal/cmd/remove"
	"github.com/exadrift/tools/arbiter/internal/cmd/restore"
	"github.com/spf13/cobra"
)

var Version string

func main() {
	if slices.Contains(os.Args, "--version") {
		fmt.Println(Version)
		return
	}

	rootCmd := &cobra.Command{
		Use: "arbiter",
	}

	rootCmd.AddCommand(initc.Command())
	rootCmd.AddCommand(bootstrap.Command())
	rootCmd.AddCommand(restore.Command())
	rootCmd.AddCommand(postconfigure.Command())
	rootCmd.AddCommand(apply.Command())
	rootCmd.AddCommand(kubeconfigcmd.Command())
	rootCmd.AddCommand(list.Command())
	rootCmd.AddCommand(remove.Command())
	_ = rootCmd.Execute()
}
