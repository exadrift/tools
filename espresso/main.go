package main

import (
	"github.com/exadrift/tools/espresso/internal/cmd/sync"
	"github.com/exadrift/tools/espresso/internal/cmd/version"
	"github.com/spf13/cobra"
)

var Version string

func main() {
	// update the version in the version package
	version.Version = Version

	rootCmd := &cobra.Command{
		Use: "espresso",
	}

	rootCmd.AddCommand(sync.Command())
	rootCmd.AddCommand(version.Command())
	_ = rootCmd.Execute()
}
