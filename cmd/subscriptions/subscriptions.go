// Package subscriptions implements the subscriptions CRUD commands for the Revenium CLI.
package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent subscriptions command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "Manage subscriptions",
	Example: `  # List all subscriptions
  revenium subscriptions list

  # Get a specific subscription
  revenium subscriptions get sub-123

  # Update with partial PATCH
  revenium subscriptions update sub-123 --patch --description "Updated"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for subscription output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Email", "Product"},
	StatusColumn: -1,
}

// toRows converts a slice of subscription maps to table row strings.
func toRows(subs []map[string]interface{}) [][]string {
	rows := make([][]string, len(subs))
	for i, s := range subs {
		rows[i] = []string{
			str(s, "id"),
			str(s, "name"),
			str(s, "label"),
			nestedStr(s, "product", "label"),
		}
	}
	return rows
}

// nestedStr extracts a string from a nested object, e.g. m["product"]["label"].
func nestedStr(m map[string]interface{}, outer, inner string) string {
	if obj, ok := m[outer].(map[string]interface{}); ok {
		return str(obj, inner)
	}
	return ""
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// renderSubscription renders a single subscription as a single-row table or JSON.
func renderSubscription(sub map[string]interface{}) error {
	rows := [][]string{{
		str(sub, "id"),
		str(sub, "name"),
		str(sub, "label"),
		nestedStr(sub, "product", "label"),
	}}
	return cmd.Output.Render(tableDef, rows, sub)
}
