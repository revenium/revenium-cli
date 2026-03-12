// Package subscribers implements the subscribers CRUD commands for the Revenium CLI.
package subscribers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent subscribers command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "subscribers",
	Short: "Manage subscribers",
	Example: `  # List all subscribers
  revenium subscribers list

  # Get a specific subscriber
  revenium subscribers get abc-123

  # Create a subscriber
  revenium subscribers create --email user@example.com`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for subscriber output.
var tableDef = output.TableDef{
	Headers: []string{"ID", "Name", "Email"},
}

// toRows converts a slice of subscriber maps to table row strings.
func toRows(subscribers []map[string]interface{}) [][]string {
	rows := make([][]string, len(subscribers))
	for i, s := range subscribers {
		name := strings.TrimSpace(str(s, "firstName") + " " + str(s, "lastName"))
		rows[i] = []string{
			str(s, "id"),
			name,
			str(s, "email"),
		}
	}
	return rows
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// renderSubscriber renders a single subscriber as a single-row table or JSON.
func renderSubscriber(sub map[string]interface{}) error {
	name := strings.TrimSpace(str(sub, "firstName") + " " + str(sub, "lastName"))
	rows := [][]string{{
		str(sub, "id"),
		name,
		str(sub, "email"),
	}}
	return cmd.Output.Render(tableDef, rows, sub)
}
