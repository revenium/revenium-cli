package metrics

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// validDimensionNames is the exact 12-value set of analytics `filter-options`
// dimension names (MTR-06, 04-RESEARCH.md § "Analytics Coverage Matrix").
// Every value maps 1:1 to GET /api/v2/analytics/filter-options/{name} on the
// analytics host.
var validDimensionNames = []string{
	"agents", "api-keys", "customers", "models", "organizations",
	"products", "providers", "teams", "tool-providers", "tools",
	"users", "vendors",
}

// validateDimensionName rejects an unknown dimension name before it ever
// reaches the request path (T-04-11 mitigation), mirroring the
// validSquadPeriods/ValidatePeriod precedent in cmd/squad_executions.go.
func validateDimensionName(c *cobra.Command, args []string) error {
	name := args[0]
	for _, v := range validDimensionNames {
		if name == v {
			return nil
		}
	}
	return fmt.Errorf("dimension %q is not valid (expected one of: %s)", name, strings.Join(validDimensionNames, ", "))
}

var dimensionsTableDef = output.TableDef{
	Headers:      []string{"Value", "Metric Result", "Metric Type"},
	StatusColumn: -1,
}

// newDimensionsCmd returns `revenium metrics dimensions <name>`, the
// analytics-backed filter-options dimension lookup family (MTR-06, D-05).
// Its PersistentPreRunE runs first, then swaps the shared cmd.APIClient onto
// the configured analytics base URL with bearer auth (D-01/D-04) — scoped to
// this command only, so sibling `metrics` leaf commands stay on the platform
// host/x-api-key auth.
func newDimensionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "dimensions <name>",
		Short: "Query analytics filter-options dimensions",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), validateDimensionName),
		Example: `  # List agent dimension values
  revenium metrics dimensions agents

  # List model dimension values
  revenium metrics dimensions models`,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// Run the root PersistentPreRunE first (config/API client init).
			// Guard root != c: when this command is executed unattached (as
			// in unit tests that call newDimensionsCmd().Execute() directly
			// without registering it under metrics.Cmd/rootCmd), c.Root()
			// returns c itself, and calling its own PersistentPreRunE would
			// recurse infinitely (stack overflow) since root.PersistentPreRunE
			// is this exact closure.
			if root := c.Root(); root != nil && root != c && root.PersistentPreRunE != nil {
				if err := root.PersistentPreRunE(c, args); err != nil {
					return err
				}
			}
			// Analytics endpoints live on a distinct, independently configured
			// host with Authorization: Bearer auth and no teamId/tenantId
			// params (D-01/D-04/D-05) — never derived by string-rewriting
			// BaseURL (Pitfall B6).
			if cmd.APIClient != nil {
				cmd.APIClient.BaseURL = cmd.APIClient.AnalyticsBaseURL
				cmd.APIClient.UseBearerAuth = true
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			var items []map[string]interface{}
			// The analytics spec's own paths already carry the full
			// /api/v2/... prefix (no buildPath/platform prefix here).
			path := "/api/v2/analytics/filter-options/" + args[0]
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &items); err != nil {
				return err
			}
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
				return nil
			}
			return cmd.Output.Render(dimensionsTableDef, toDimensionRows(items), items)
		},
	}

	cmd.AddListFlags(c)
	return c
}

func toDimensionRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "value"),
			formatNumber(floatVal(m, "metricResult")),
			str(m, "metricType"),
		}
	}
	return rows
}
