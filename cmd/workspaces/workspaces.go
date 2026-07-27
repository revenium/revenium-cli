// Package workspaces implements the read-only workspaces command for the
// Revenium CLI (list only — D-01). The live API spec has no
// GET /v2/api/workspaces/{id} endpoint, only list + a mutating rename
// (PATCH .../name, out of scope per D-02). See 06-RESEARCH.md Open
// Question #5 / 06-CONTEXT.md D-01 for the documented get-gap, flagged for
// the Phase 8 drift report.
package workspaces

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent workspaces command, exported for registration in
// main.go (registration deferred to 06-06 per plan).
var Cmd = &cobra.Command{
	Use:   "workspaces",
	Short: "List workspaces",
	Example: `  # List all workspaces
  revenium workspaces list

  # Filter by provider
  revenium workspaces list --provider aws

  # Filter by query
  revenium workspaces list --query prod`,
}

func init() {
	// D-01: list ONLY. There is deliberately no `get <id>` here — no such
	// endpoint exists in the live spec. Do not fabricate one by
	// re-fetching+filtering the list (footgun: silently expensive, gives a
	// false impression a real single-resource endpoint exists).
	Cmd.AddCommand(newListCmd())
}

// tableDef defines the 3-column layout for the workspaces list verb.
// WorkspaceMetadataResource has no natural lifecycle status field, so
// StatusColumn is -1 (matches cmd/squads/list.go's listTableDef precedent).
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Provider"},
	StatusColumn: -1,
}

// str safely extracts a string value from a map, returning "" for missing
// or nil keys. Package-local copy (established duplication pattern across
// cmd/organizations, cmd/squads, etc. — see 06-PATTERNS.md).
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// toRows converts a slice of workspace maps to 3-col table row strings.
func toRows(workspaces []map[string]interface{}) [][]string {
	rows := make([][]string, len(workspaces))
	for i, w := range workspaces {
		rows[i] = []string{str(w, "id"), str(w, "name"), str(w, "provider")}
	}
	return rows
}
