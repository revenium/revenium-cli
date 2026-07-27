package models

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// revisionTableDef defines the table layout for a model's audit/revision
// history (RES-01). Local to this package per CONVENTIONS.md's per-package
// render-helper norm — this identical shape is intentionally duplicated in
// cmd/products/history.go and cmd/sources/history.go rather than shared, to
// avoid a resource-to-resource package import.
var revisionTableDef = output.TableDef{
	Headers:      []string{"Attribute", "Field", "Change Type", "Previous Value", "Current Value", "Changed", "User"},
	StatusColumn: -1,
}

// toRevisionRows converts a slice of RevisionDataResource_Read maps to table rows.
func toRevisionRows(revisions []map[string]interface{}) [][]string {
	rows := make([][]string, len(revisions))
	for i, r := range revisions {
		rows[i] = []string{
			str(r, "attribute"),
			str(r, "field"),
			str(r, "changeType"),
			str(r, "previousValue"),
			str(r, "currentValue"),
			str(r, "changeDate"),
			str(r, "user"),
		}
	}
	return rows
}

// newHistoryCmd returns the `revenium models history <id>` command.
//
// RES-01: GETs /v2/api/sources/ai/models/{id}/revisions (the roadmap verb is
// "history"; the spec's path segment is "revisions" per D-08 naming discretion).
// Renders RevisionDataResource_Read rows.
func newHistoryCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "history <id>",
		Short: "List the audit/change history for an AI model",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List a model's change history
  revenium models history abc-123

  # As JSON
  revenium models history abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/revisions", url.PathEscape(args[0]))
			var items []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &items); err != nil {
				return err
			}
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No history found.")
				return nil
			}
			return cmd.Output.Render(revisionTableDef, toRevisionRows(items), items)
		},
	}

	cmd.AddListFlags(c)
	return c
}
