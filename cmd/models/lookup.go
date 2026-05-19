package models

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newLookupCmd returns the `lookup --name <name>` sub-command. Reads the
// OAS-LOCKED Candidate A path-templated endpoint (per Phase 16 RESEARCH D-01 —
// name is interpolated into the URL path segment, NOT a query parameter, so
// url.PathEscape is required). 404 responses flow through the default
// mapHTTPError path producing "Resource not found." (D-03 — no per-verb
// override). Missing --name is rejected by Cobra MarkFlagRequired before any
// HTTP round-trip (D-04).
func newLookupCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "lookup",
		Short: "Look up an AI model by name",
		Args:  cobra.NoArgs,
		Example: `  # Look up a model by name
  revenium models lookup --name gpt-4

  # As JSON
  revenium models lookup --name gpt-4 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/sources/ai/models/name/%s", url.PathEscape(name))
			var model map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &model); err != nil {
				return err
			}
			return renderModel(model)
		},
	}
	c.Flags().StringVar(&name, "name", "", "Name of the AI model to look up")
	_ = c.MarkFlagRequired("name")
	return c
}
