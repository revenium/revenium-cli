// Package users implements the users CRUD commands for the Revenium CLI.
package users

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent users command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Example: `  # List all users
  revenium users list

  # Get a specific user
  revenium users get user-123

  # Create a user
  revenium users create --email jane@example.com --first-name Jane --last-name Doe --roles ROLE_API_CONSUMER --team-ids team-1`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for user output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Email", "Name", "Roles"},
	StatusColumn: -1,
}

// toRows converts a slice of user maps to table row strings.
func toRows(users []map[string]interface{}) [][]string {
	rows := make([][]string, len(users))
	for i, u := range users {
		name := strings.TrimSpace(str(u, "firstName") + " " + str(u, "lastName"))
		rows[i] = []string{
			str(u, "id"),
			str(u, "email"),
			name,
			rolesStr(u),
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

// rolesStr extracts the roles field and joins elements with ", ".
func rolesStr(m map[string]interface{}) string {
	v, ok := m["roles"]
	if !ok || v == nil {
		return ""
	}
	arr, ok := v.([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, len(arr))
	for i, el := range arr {
		parts[i] = fmt.Sprint(el)
	}
	return strings.Join(parts, ", ")
}

// renderUser renders a single user as a single-row table or JSON.
func renderUser(user map[string]interface{}) error {
	name := strings.TrimSpace(str(user, "firstName") + " " + str(user, "lastName"))
	rows := [][]string{{
		str(user, "id"),
		str(user, "email"),
		name,
		rolesStr(user),
	}}
	return cmd.Output.Render(tableDef, rows, user)
}
