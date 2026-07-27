package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// readBulkPricingInput reads a JSON array of pricing dimension entries from a
// file path, or from stdin when path is "-". No existing codebase precedent
// for file/stdin JSON-array input — built fresh per 05-RESEARCH.md's Code
// Examples section (D-07).
func readBulkPricingInput(path string) ([]map[string]interface{}, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read bulk pricing input: %w", err)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse bulk pricing input as a JSON array: %w", err)
	}
	return entries, nil
}

// renderBulkSaveResult renders the bulk-save PUT response. Per 05-RESEARCH.md's
// A2 open question, the exact success shape (an array of
// PricingDimensionResource_Read, or a PricingResponse_Read wrapper with a
// "dimensions" key) wasn't confirmed pre-implementation, so this tolerates
// both shapes and falls back to raw JSON for anything else.
func renderBulkSaveResult(result interface{}) error {
	toMapSlice := func(items []interface{}) []map[string]interface{} {
		out := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	}

	switch v := result.(type) {
	case []interface{}:
		dims := toMapSlice(v)
		if len(dims) == 0 {
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON([]interface{}{})
			}
			fmt.Fprintln(cmd.Output.Writer(), "No pricing dimensions returned.")
			return nil
		}
		return cmd.Output.Render(pricingTableDef, toPricingRows(dims), dims)
	case map[string]interface{}:
		if dims, ok := v["dimensions"].([]interface{}); ok {
			converted := toMapSlice(dims)
			return cmd.Output.Render(pricingTableDef, toPricingRows(converted), converted)
		}
		return cmd.Output.RenderJSON(v)
	default:
		return cmd.Output.RenderJSON(v)
	}
}

// newPricingBulkSaveCmd returns the `revenium models pricing bulk-save <model-id>` command.
//
// RES-02/D-07: PUTs a bare JSON array to
// /v2/api/sources/ai/models/{modelId}/pricing/dimensions
// (batch_save_pricing_dimensions). This is a destructive full-replace — the
// spec states existing dimensions not present in the list are deleted.
// Uses Do() directly, not the GET-merge-PUT update helper — no merge
// semantics apply to a collection-level array PUT.
func newPricingBulkSaveCmd() *cobra.Command {
	var file string

	c := &cobra.Command{
		Use:   "bulk-save <model-id>",
		Short: "Replace ALL pricing dimensions for a model from a JSON file or stdin",
		Long: `Replace ALL pricing dimensions for a model from a JSON file or stdin.

WARNING: this is a destructive full-replace operation. Any existing pricing
dimensions for the model that are NOT present in the supplied JSON array will
be deleted. Use --dry-run to preview the full replacement set before applying it.`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Replace all pricing dimensions from a file
  revenium models pricing bulk-save abc-123 --file dimensions.json

  # Replace all pricing dimensions from stdin
  cat dimensions.json | revenium models pricing bulk-save abc-123 --file -

  # Preview the replacement set without applying it
  revenium models pricing bulk-save abc-123 --file dimensions.json --dry-run`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			entries, err := readBulkPricingInput(file)
			if err != nil {
				return err
			}

			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions", url.PathEscape(args[0]))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "bulk-save", "pricing dimensions", path, entries)
			}

			var result interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, entries, &result); err != nil {
				return err
			}
			return renderBulkSaveResult(result)
		},
	}

	c.Flags().StringVar(&file, "file", "", "Path to a JSON array file, or - to read from stdin (required)")
	_ = c.MarkFlagRequired("file")

	return c
}
