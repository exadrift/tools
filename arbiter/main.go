package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/exadrift/tools/arbiter/internal/cmd/bootstrap"
	"github.com/exadrift/tools/arbiter/internal/cmd/initc"
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
	_ = rootCmd.Execute()
}
