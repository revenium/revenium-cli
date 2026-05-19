package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newTransactionsCmd()) }

// transactionsTableDef defines the 4-column transactions table layout
// (RESEARCH §A3). StatusColumn=3 dispatches the Status cell through
// statusStyle, which Plan 13-01 extended to color lowercase "error" red.
var transactionsTableDef = output.TableDef{
	Headers:      []string{"Timestamp", "Model", "Cost", "Status"},
	StatusColumn: 3,
}

// newTransactionsCmd returns the `revenium jobs transactions <agenticJobId>`
// subcommand. It GETs /v2/api/jobs/<id>/transactions, decodes the wrapper
// object {transactions: [...], totalCount: N} — the response is NOT Spring
// HATEOAS, so the auto-unwrap list helper is intentionally bypassed per
// RESEARCH §A3 + Pitfall 2 — and renders the array as a 4-column table.
// Empty arrays render "No transactions found." (stdout) or the full wrapper
// {"transactions":[],"totalCount":0} (--json) to preserve totalCount for
// script consumers (RESEARCH §A3).
func newTransactionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transactions <agenticJobId>",
		Short: "List a job's AI transactions",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List transactions for a job
  revenium jobs transactions loan-app-12345

  # List transactions as JSON (preserves totalCount wrapper)
  revenium jobs transactions loan-app-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/jobs/%s/transactions", url.PathEscape(id))

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}

			// Extract transactions slice from the wrapper object. Per RESEARCH §A3,
			// the response is NOT Spring HATEOAS — it is a flat wrapper object
			// {transactions: [...], totalCount: N}. We type-assert defensively:
			// mis-typed elements are silently skipped (T-13-04-02 mitigation).
			raw, _ := resp["transactions"].([]interface{})
			txns := make([]map[string]interface{}, 0, len(raw))
			for _, r := range raw {
				if m, ok := r.(map[string]interface{}); ok {
					txns = append(txns, m)
				}
			}

			if len(txns) == 0 {
				if cmd.Output.IsJSON() {
					// Preserve the wrapper {"transactions":[],"totalCount":0} for
					// script consumers reading totalCount (RESEARCH §A3 / T-13-04-04).
					// This diverges from list.go which emits a bare [].
					return cmd.Output.RenderJSON(resp)
				}
				fmt.Fprintln(c.OutOrStdout(), "No transactions found.")
				return nil
			}

			rows := make([][]string, len(txns))
			for i, t := range txns {
				rows[i] = []string{
					str(t, "timestamp"),
					str(t, "model"),
					// Cost is always USD per OAS (RESEARCH §A3). Uses Plan 13-01's
					// extracted helpers — no local formatCurrency or floatVal.
					output.FormatCurrency(output.FloatVal(t, "cost"), "USD"),
					str(t, "status"),
				}
			}
			// Render's third arg is the full wrapper resp (NOT txns) so that
			// --json emits the wrapper object including totalCount.
			return cmd.Output.Render(transactionsTableDef, rows, resp)
		},
	}
}
