package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/exadrift/tools/arbiter/internal/cmd/bootstrap"
	"github.com/exadrift/tools/arbiter/internal/cmd/initc"
	postconfigure "github.com/exadrift/tools/arbiter/internal/cmd/post-configure"
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
	_ = rootCmd.Execute()
}
