package models

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// pricingCoverageTableDef defines the key-value table layout for pricing
// coverage output.
var pricingCoverageTableDef = output.TableDef{
	Headers:      []string{"Field", "Value"},
	StatusColumn: -1,
}

// renderPricingCoverage renders the 6-boolean PricingCoverageResource_Read
// shape as a key-value table or JSON. Mirrors renderPromptSettings in
// cmd/teams/prompt_capture.go.
func renderPricingCoverage(coverage map[string]interface{}) error {
	var rows [][]string
	for key, val := range coverage {
		if key == "_links" {
			continue
		}
		rows = append(rows, []string{key, fmt.Sprint(val)})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})
	return cmd.Output.Render(pricingCoverageTableDef, rows, coverage)
}

// newPricingCoverageCmd returns the `revenium models pricing coverage <model-id>` command.
//
// RES-02: GETs /v2/api/sources/ai/models/{modelId}/pricing/coverage, returning
// PricingCoverageResource_Read's 6 flat booleans (hasAudioPricing,
// hasCharacterPricing, hasCreditsPricing, hasImagePricing, hasTokenPricing,
// hasVideoPricing).
func newPricingCoverageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "coverage <model-id>",
		Short: "Show which pricing dimension types are covered for a model",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Show pricing coverage for a model
  revenium models pricing coverage abc-123

  # As JSON
  revenium models pricing coverage abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/coverage", url.PathEscape(args[0]))
			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resp); err != nil {
				return err
			}
			return renderPricingCoverage(resp)
		},
	}
}
