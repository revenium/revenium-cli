package anomalies

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// dimensionsPath is the GET endpoint for the anomaly provider-dimensions
// lookup: GET /v2/api/sources/ai/anomaly/provider-dimensions
// (getProviderDimensions).
const dimensionsPath = "/v2/api/sources/ai/anomaly/provider-dimensions"

// dimensionsTableDef defines the table layout for the anomalies dimensions
// counts summary. The full ProviderDimensionsResponse is deeply nested
// (arrays of objects, maps) and would render as ugly Go literals via a
// generic fmt.Sprint key-value renderer (see 05-PATTERNS.md) — table mode
// intentionally shows only a top-level counts summary; use --json for the
// full nested response. apiKeys are masked hints server-side; this command
// never renders raw secrets in table mode (T-05-09).
var dimensionsTableDef = output.TableDef{
	Headers:      []string{"# Providers", "# Workspaces", "# Models", "# Supported Dimensions"},
	StatusColumn: -1,
}

// newDimensionsCmd returns the `revenium anomalies dimensions` command.
func newDimensionsCmd() *cobra.Command {
	var provider string

	c := &cobra.Command{
		Use:   "dimensions",
		Short: "Get available anomaly provider dimensions (providers, workspaces, models)",
		Args:  cobra.NoArgs,
		Example: `  # Get provider dimensions counts summary
  revenium anomalies dimensions

  # Filter by provider
  revenium anomalies dimensions --provider anthropic

  # Get the full nested response as JSON
  revenium anomalies dimensions --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := dimensionsPath
			if c.Flags().Changed("provider") {
				path += "?provider=" + url.QueryEscape(provider)
			}

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}

			if !cmd.Output.IsJSON() {
				fmt.Fprintln(c.OutOrStdout(), "(use --json for full detail, including provider/workspace/model names)")
			}

			rows := [][]string{{
				countStr(resp, "providers"),
				countStr(resp, "workspaces"),
				countStr(resp, "models"),
				countStr(resp, "supportedDimensions"),
			}}
			return cmd.Output.Render(dimensionsTableDef, rows, resp)
		},
	}

	c.Flags().StringVar(&provider, "provider", "", "Filter by provider")
	return c
}

// countStr returns the length of a []interface{} value at key as a string,
// or "0" if the key is absent or not an array.
func countStr(m map[string]interface{}, key string) string {
	if arr, ok := m[key].([]interface{}); ok {
		return fmt.Sprintf("%d", len(arr))
	}
	return "0"
}
