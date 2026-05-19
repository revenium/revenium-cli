// Package organizations implements the organizations CRUD commands for the Revenium CLI.
package organizations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent organizations command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "organizations",
	Short: "Manage organizations",
	Example: `  # List all organizations
  revenium organizations list

  # Get a specific organization
  revenium organizations get org-123

  # Create an organization
  revenium organizations create --name "Acme Corporation"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
	Cmd.AddCommand(newTagsCmd())
	Cmd.AddCommand(newChildrenCmd())
}

// tableDef defines the shared 3-column layout for list/get/children verbs (D-04).
// Strict mirror of cmd/jobs/jobs.go — Status column is index 2 (zero-indexed).
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
}

// tagsTableDef defines the single-column layout for the tags verb (D-02 Candidate A
// per RESEARCH — the OAS get_organization_tags response shape is []string).
var tagsTableDef = output.TableDef{
	Headers:      []string{"Tag"},
	StatusColumn: -1,
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// orgStatus returns the Status-column value for an organization. Per RESEARCH
// D-04a LOCKED, OrganizationResource_Read has no natural lifecycle status field
// (no `status`, no `state` enum, no `enabled` bool — the AutoDiscoveryEnabled
// booleans are feature flags, not lifecycle). Empty cell is the documented
// fallback (CONTEXT.md D-04a / Phase 12 D-11 / Phase 14 D-03 precedent). The
// function is defined for future extensibility — if a status-like field appears
// in a future OAS revision, point it at the new field here.
func orgStatus(m map[string]interface{}) string {
	return ""
}

// toRows converts a slice of organization maps to 3-col table row strings.
// Shared by the list and children verbs (D-03 / D-04c).
func toRows(orgs []map[string]interface{}) [][]string {
	rows := make([][]string, len(orgs))
	for i, o := range orgs {
		name := str(o, "name")
		if name == "" {
			name = str(o, "label")
		}
		rows[i] = []string{
			str(o, "id"),
			name,
			orgStatus(o),
		}
	}
	return rows
}

// renderOrg renders a single organization as a single-row table or JSON.
// Shared by the get / create / update verbs (D-04c).
func renderOrg(org map[string]interface{}) error {
	name := str(org, "name")
	if name == "" {
		name = str(org, "label")
	}
	rows := [][]string{{
		str(org, "id"),
		name,
		orgStatus(org),
	}}
	return cmd.Output.Render(tableDef, rows, org)
}
