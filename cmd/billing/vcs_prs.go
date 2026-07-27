package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newVcsPrsCmd()) }

// vcsPrsTableDef defines the table layout for the VCS PR/MR summary. Full
// response available via --json (dimensions.go idiom).
var vcsPrsTableDef = output.TableDef{
	Headers:      []string{"# Team Members", "Total PRs"},
	StatusColumn: -1,
}

// newVcsPrsCmd returns `revenium billing vcs-prs` (GET
// /v2/api/billing/users/vcs-prs, single-object report — exact analog to
// cmd/anomalies/dimensions.go's raw-Do-into-map idiom).
func newVcsPrsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vcs-prs",
		Short: "Get PR/MR summary by team member",
		Args:  cobra.NoArgs,
		Example: `  # Get PR/MR summary by team member
  revenium billing vcs-prs

  # Get the full response as JSON
  revenium billing vcs-prs --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/users/vcs-prs", nil, &resp); err != nil {
				return err
			}

			if !cmd.Output.IsJSON() {
				fmt.Fprintln(c.OutOrStdout(), "(use --json for full detail)")
			}

			rows := [][]string{{
				countStr(resp, "summaries"),
				formatCount(floatVal(resp, "totalPRs")),
			}}
			return cmd.Output.Render(vcsPrsTableDef, rows, resp)
		},
	}
}
