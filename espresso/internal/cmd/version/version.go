package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "none"

func Command() *cobra.Command {
	base := &cobra.Command{
		Use:   "version",
		Short: "display the tool version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	return base
}
