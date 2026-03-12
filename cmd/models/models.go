// Package models implements the AI model CRUD commands for the Revenium CLI.
package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent models command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "models",
	Short: "Manage AI models and pricing",
	Example: `  # List all AI models
  revenium models list

  # Get a specific model
  revenium models get abc-123`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// modelTableDef defines the table layout for model output.
var modelTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Provider", "Mode"},
	StatusColumn: -1,
}

// toModelRows converts a slice of model maps to table row strings.
func toModelRows(models []map[string]interface{}) [][]string {
	rows := make([][]string, len(models))
	for i, m := range models {
		rows[i] = []string{
			str(m, "id"),
			str(m, "name"),
			str(m, "provider"),
			str(m, "mode"),
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

// renderModel renders a single model as a single-row table or JSON.
func renderModel(model map[string]interface{}) error {
	rows := [][]string{{
		str(model, "id"),
		str(model, "name"),
		str(model, "provider"),
		str(model, "mode"),
	}}
	return cmd.Output.Render(modelTableDef, rows, model)
}
