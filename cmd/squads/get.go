package squads

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newGetCmd()) }

// getTableDef defines the squad-detail table layout (SQUAD-07). Columns
// follow the real SquadDetailResource_Read fields.
var getTableDef = output.TableDef{
	Headers:      []string{"ID", "Squad", "Agents", "Traces", "Transactions", "Errors", "Duration", "Cost", "Status"},
	StatusColumn: 8,
}

// renderSquad renders a single squad detail as a single-row table or JSON.
func renderSquad(squad map[string]interface{}) error {
	rows := [][]string{{
		str(squad, "id"),
		str(squad, "squadName"),
		formatNumber(floatVal(squad, "agentCount")),
		formatNumber(floatVal(squad, "traceCount")),
		formatNumber(floatVal(squad, "transactionCount")),
		formatNumber(floatVal(squad, "errorCount")),
		formatDuration(floatVal(squad, "duration")),
		formatCost(floatVal(squad, "totalCost")),
		str(squad, "status"),
	}}
	return cmd.Output.Render(getTableDef, rows, squad)
}

// newGetCmd returns the `revenium squads get <squadId>` subcommand
// (GET /v2/api/squads/{squadId}, SQUAD-07, D-02). The positional squadId is
// the raw telemetry squadId identity space (RESEARCH), distinct from the
// hash-encoded id used by `squads list`. T-2-01 mitigation:
// cmd.ValidResourceID rejects path traversal/control chars before
// url.PathEscape interpolates the id into the request path.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <squadId>",
		Short: "Get a squad's detail by squadId",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a squad's detail
  revenium squads get squad-loan-proc-12345

  # Get a squad's detail for a specific period
  revenium squads get squad-loan-proc-12345 --period SEVEN_DAYS

  # Get a squad's detail as JSON
  revenium squads get squad-loan-proc-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := buildSquadsPath(fmt.Sprintf("/v2/api/squads/%s", url.PathEscape(args[0])))
			var squad map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &squad); err != nil {
				return err
			}
			return renderSquad(squad)
		},
	}
}
