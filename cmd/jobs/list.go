package jobs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List all jobs",
		Args:  cobra.NoArgs,
		Example: `  # List all jobs
  revenium jobs list

  # List jobs as JSON
  revenium jobs list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var jobs []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/jobs", cmd.ListOptsFromFlags(c), &jobs); err != nil {
				return err
			}
			if len(jobs) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No jobs found.")
				return nil
			}
			return cmd.Output.Render(tableDef, toRows(jobs), jobs)
		},
	}

	cmd.AddListFlags(c)
	return c
}
