package billing

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newClaudeCodeContributionsCmd()) }

// claudeCodeContributionsTableDef defines the table layout for the
// Claude Code contributions summary. Full response available via --json
// (dimensions.go idiom).
var claudeCodeContributionsTableDef = output.TableDef{
	Headers:      []string{"# Contributors", "Total Contributions"},
	StatusColumn: -1,
}

// newClaudeCodeContributionsCmd returns `revenium billing
// claude-code-contributions` (GET
// /v2/api/billing/users/claude-code-contributions, single-object report —
// exact analog to cmd/anomalies/dimensions.go's raw-Do-into-map idiom).
//
// READ-ONLY ONLY (CONTEXT.md D-02): the sibling mutating verb
// `POST /v2/api/billing/users/claude-code-contributions/sync`
// (syncClaudeCodeContributions) is explicitly EXCLUDED from this
// read-only phase. Do not add a `sync` subcommand here.
func newClaudeCodeContributionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claude-code-contributions",
		Short: "Get Claude Code contribution attribution",
		Args:  cobra.NoArgs,
		Example: `  # Get Claude Code contribution summary
  revenium billing claude-code-contributions

  # Get the full response as JSON
  revenium billing claude-code-contributions --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/billing/users/claude-code-contributions", nil, &resp); err != nil {
				return err
			}

			if !cmd.Output.IsJSON() {
				fmt.Fprintln(c.OutOrStdout(), "(use --json for full detail)")
			}

			rows := [][]string{{
				countStr(resp, "contributions"),
				formatCount(floatVal(resp, "totalContributions")),
			}}
			return cmd.Output.Render(claudeCodeContributionsTableDef, rows, resp)
		},
	}
}
