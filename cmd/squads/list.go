package squads

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newListCmd()) }

// listTableDef defines the squad-group table layout (SQUAD-06). Columns
// follow the real EntityModelSquadResource_Read fields; no status column
// (a squad group has success/partial/failed counts, not a single status).
var listTableDef = output.TableDef{
	Headers:      []string{"ID", "Label", "Source", "Executions", "Agents", "Traces", "Cost", "Success", "Partial", "Failed"},
	StatusColumn: -1,
}

// toListRows converts a slice of squad-group maps to table row strings.
func toListRows(squads []map[string]interface{}) [][]string {
	rows := make([][]string, len(squads))
	for i, s := range squads {
		rows[i] = []string{
			str(s, "id"),
			str(s, "label"),
			str(s, "source"),
			formatNumber(floatVal(s, "executionCount")),
			formatNumber(floatVal(s, "agentCount")),
			formatNumber(floatVal(s, "traceCount")),
			formatCost(floatVal(s, "totalCost")),
			formatNumber(floatVal(s, "successCount")),
			formatNumber(floatVal(s, "partialCount")),
			formatNumber(floatVal(s, "failedCount")),
		}
	}
	return rows
}

// newListCmd returns the `revenium squads list` subcommand
// (GET /v2/api/squads/entities, SQUAD-06, D-02). teamId is auto-injected by
// cmd.APIClient.DoList — no --team-id flag needed.
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List squad groups",
		Args:  cobra.NoArgs,
		Example: `  # List squad groups
  revenium squads list

  # List squad groups for the last 7 days
  revenium squads list --period SEVEN_DAYS

  # List squad groups as JSON
  revenium squads list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var squads []map[string]interface{}
			path := buildSquadsPath("/v2/api/squads/entities")
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &squads); err != nil {
				return err
			}
			if len(squads) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No squads found.")
				return nil
			}
			return cmd.Output.Render(listTableDef, toListRows(squads), squads)
		},
	}

	cmd.AddListFlags(c)
	return c
}
