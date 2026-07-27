// Package meteringelements implements the metering-elements CRUD commands
// for the Revenium CLI.
//
// SAFE-01 triple-split (main_test.go TestAllCommandPackagesRegistered):
// directory cmd/metering-elements/ (hyphenated, matches Cmd.Use's first
// token) / Go package identifier meteringelements (no hyphen — Go syntax
// rule) / Cmd.Use: "metering-elements". See 06-PATTERNS.md "SAFE-01 naming
// triple-split".
package meteringelements

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent metering-elements command, exported for registration in
// main.go (deferred to 06-06 per this plan's objective).
var Cmd = &cobra.Command{
	Use:   "metering-elements",
	Short: "Manage metering element definitions",
	Example: `  # List all metering element definitions
  revenium metering-elements list

  # Get a specific metering element definition
  revenium metering-elements get me-123

  # Create a metering element definition
  revenium metering-elements create --name "Tokens" --type NUMBER`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the shared 4-column layout for list/get verbs.
// MeteringElementDefinitionResource_Read has no lifecycle status field
// (same precedent as cmd/organizations' orgStatus — D-04a), so there is no
// status column: StatusColumn: -1.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Type", "Description"},
	StatusColumn: -1,
}

// str safely extracts a string value from a map, returning "" for missing
// or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// toRows converts a slice of metering element maps to 4-col table row
// strings.
func toRows(elements []map[string]interface{}) [][]string {
	rows := make([][]string, len(elements))
	for i, e := range elements {
		rows[i] = []string{
			str(e, "id"),
			str(e, "name"),
			str(e, "type"),
			str(e, "description"),
		}
	}
	return rows
}

// renderElement renders a single metering element definition as a
// single-row table or JSON. Shared by the get / create / update verbs.
func renderElement(element map[string]interface{}) error {
	rows := [][]string{{
		str(element, "id"),
		str(element, "name"),
		str(element, "type"),
		str(element, "description"),
	}}
	return cmd.Output.Render(tableDef, rows, element)
}

// validTypes enumerates the allowed `type` values for a metering element
// definition (components.schemas.MeteringElementDefinitionResource_Write.
// properties.type.enum, verified against the live OpenAPI spec).
var validTypes = []string{"STRING", "NUMBER"}

// validateType enforces the type enum client-side before any dry-run
// render or HTTP call (T-06-13 mitigation). Shared by create and update.
func validateType(t string) error {
	for _, v := range validTypes {
		if t == v {
			return nil
		}
	}
	return fmt.Errorf("type %q is not valid (expected STRING or NUMBER)", t)
}
