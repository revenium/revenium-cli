package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/build"
)

// newVersionCmd creates the version subcommand.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Example: `  # Print version
  revenium version`,
		Run: func(cmd *cobra.Command, args []string) {
			commit := build.Commit
			if len(commit) > 7 {
				commit = commit[:7]
			}
			fmt.Fprintf(cmd.OutOrStdout(), "revenium %s (%s)\n", build.Version, commit)
		},
	}
}
