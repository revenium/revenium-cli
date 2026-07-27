// Package tenants implements the read-only tenants commands for the
// Revenium CLI (get + usage only — D-03). The newly-discovered
// ingestion-failures/strict-ingestion-mode/join-requests sub-resources are
// deliberately excluded (spec drift, flagged for the Phase 8 drift report),
// and the payment-method default toggle is out of scope per PROJECT.md.
package tenants

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent tenants command, exported for registration in main.go
// (registration deferred to 06-06 per plan).
var Cmd = &cobra.Command{
	Use:   "tenants",
	Short: "Inspect tenants",
	Example: `  # Get a tenant by ID
  revenium tenants get ten-123

  # Get a tenant's usage summary
  revenium tenants usage ten-123`,
}

func init() {
	// D-03: get + usage ONLY. No ingestion-failures/strict-ingestion-mode/
	// join-requests (spec-drift, not yet triaged — see 06-RESEARCH.md
	// Pitfall 5) and no payment-method toggle (Out of Scope, PROJECT.md).
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newUsageCmd())
}

// tenantsTableDef defines the 3-column layout for the tenants get verb.
var tenantsTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
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

// renderTenant renders a single tenant as a single-row table or JSON.
func renderTenant(tenant map[string]interface{}) error {
	rows := [][]string{{
		str(tenant, "id"),
		str(tenant, "name"),
		str(tenant, "status"),
	}}
	return cmd.Output.Render(tenantsTableDef, rows, tenant)
}
