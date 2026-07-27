package tools

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newLookupCmd returns the `revenium tools lookup --tool-id <toolId>` command.
//
// RES-09 scope correction: the requirement text says "lookup by name/identifier",
// but the live spec only supports lookup by the external `toolId` — there is no
// name-based lookup operation (the `tools list` `query` param is a generic search
// filter, not a guaranteed-single-match name lookup like `models lookup --name`).
// This command intentionally has no `--name` flag; only `--tool-id` is supported.
func newLookupCmd() *cobra.Command {
	var toolID string

	c := &cobra.Command{
		Use:   "lookup",
		Short: "Look up a tool by its external tool ID",
		Args:  cobra.NoArgs,
		Example: `  # Look up a tool by its external tool ID
  revenium tools lookup --tool-id my-tool

  # As JSON
  revenium tools lookup --tool-id my-tool --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// toolId is a PATH param, not a query param (differs from users lookup's
			// query-string shape) — use url.PathEscape, not url.QueryEscape.
			path := fmt.Sprintf("/v2/api/tools/by-tool-id/%s", url.PathEscape(toolID))
			var tool map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &tool); err != nil {
				return err
			}
			return renderTool(tool)
		},
	}

	c.Flags().StringVar(&toolID, "tool-id", "", "External tool ID to look up (distinct from the internal id used by 'tools get')")
	_ = c.MarkFlagRequired("tool-id")

	return c
}
