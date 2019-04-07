package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version creates the Cobra Command For The Updater Action
func Version(GitCommit string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get the current version of Netlify Dynamic DNS.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("GitCommit: ", GitCommit)
			return nil
		},
	}
}
