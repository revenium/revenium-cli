package jobs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// init registers newTypesCmd() onto the package-level jobs.Cmd. Per Phase 12
// D-21 and Go's multi-init-per-package semantics, this composes with the
// init()s in jobs.go (list+get) and create.go/update.go/delete.go/outcome.go.
func init() {
	Cmd.AddCommand(newTypesCmd())
}

// typesTableDef defines the single-column layout for the types output.
// The "Type" header already maps to categoryStyle in internal/output/styles.go
// (line 50), so no style extension is required. StatusColumn = -1 disables
// status colorization — type names are not status tokens.
var typesTableDef = output.TableDef{
	Headers:      []string{"Type"},
	StatusColumn: -1,
}

// newTypesCmd builds `revenium jobs types` — JOBS-09.
//
// The OAS surface (GET /v2/api/jobs/types) returns a bare []string, NOT a
// Spring HATEOAS list — the typed decode target is var types []string per
// RESEARCH §A4 + Pitfall 2 (the auto-pagination helper does not fit this
// shape because it expects *[]map[string]interface{}).
//
// Per CF-17, aggregate verbs (no id segment) use cobra.NoArgs.
func newTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List available job types",
		Args:  cobra.NoArgs, // CF-17 — aggregate verb, no id
		Example: `  # List all available job types
  revenium jobs types

  # As JSON for scripting
  revenium jobs types --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// Typed []string decode — pagination helper does not fit
			// this shape (RESEARCH §A4 + Pitfall 2).
			var types []string
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/jobs/types", nil, &types); err != nil {
				return err
			}

			// Empty-state branching per Phase 12 D-09 (cmd/jobs/list.go:26-32),
			// using a typed []string{} so --json emits a typed empty array.
			if len(types) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]string{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No job types found.")
				return nil
			}

			rows := make([][]string, len(types))
			for i, t := range types {
				rows[i] = []string{t}
			}
			// Pass the raw []string as the third arg so --json emits the
			// unmodified server response (not the table rows).
			return cmd.Output.Render(typesTableDef, rows, types)
		},
	}
}
