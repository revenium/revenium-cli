// Package teams implements the teams CRUD commands for the Revenium CLI.
package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent teams command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "teams",
	Short: "Manage teams",
	Example: `  # List all teams
  revenium teams list

  # Get a specific team
  revenium teams get team-123

  # Create a team
  revenium teams create --name "Engineering"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
	Cmd.AddCommand(promptCaptureCmd)
	initPromptCapture()
}

// tableDef defines the table layout for team output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name"},
	StatusColumn: -1,
}

// toRows converts a slice of team maps to table row strings.
func toRows(teams []map[string]interface{}) [][]string {
	rows := make([][]string, len(teams))
	for i, t := range teams {
		name := str(t, "label")
		if name == "" {
			name = str(t, "name")
		}
		rows[i] = []string{
			str(t, "id"),
			name,
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

// renderTeam renders a single team as a single-row table or JSON.
func renderTeam(team map[string]interface{}) error {
	name := str(team, "label")
	if name == "" {
		name = str(team, "name")
	}
	rows := [][]string{{
		str(team, "id"),
		name,
	}}
	return cmd.Output.Render(tableDef, rows, team)
}
