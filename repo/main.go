package main

import (
	"log"

	"github.com/exadrift/tools/repo/internal/cmd/branch"
	"github.com/exadrift/tools/repo/internal/cmd/prompt"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "repo",
	}

	rootCmd.AddCommand(branch.Command())
	rootCmd.AddCommand(prompt.Command())
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
