// Package credentials implements the credentials CRUD commands for the Revenium CLI.
package credentials

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent credentials command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "credentials",
	Short: "Manage provider credentials",
	Example: `  # List all credentials
  revenium credentials list

  # Get a specific credential
  revenium credentials get cred-123

  # Create a credential
  revenium credentials create --label "My API Key" --api-key "sk-abc123"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for credential output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Label", "Provider", "Type", "Secret"},
	StatusColumn: -1,
}

// toRows converts a slice of credential maps to table row strings.
func toRows(credentials []map[string]interface{}) [][]string {
	rows := make([][]string, len(credentials))
	for i, c := range credentials {
		rows[i] = []string{
			str(c, "id"),
			str(c, "label"),
			str(c, "provider"),
			str(c, "credentialType"),
			maskSecret(str(c, "apiKey")),
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

// maskSecret masks a secret value for display. It preserves any prefix before
// the first hyphen and always shows the last 4 characters.
// Examples: "sk-abc123xyz7f3a" -> "sk-****7f3a", "short" -> "****hort", "" -> ""
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****" + s
	}
	idx := strings.Index(s, "-")
	if idx >= 0 && idx < len(s)-4 {
		return s[:idx+1] + "****" + s[len(s)-4:]
	}
	return "****" + s[len(s)-4:]
}

// renderCredential renders a single credential as a single-row table or JSON.
func renderCredential(credential map[string]interface{}) error {
	rows := [][]string{{
		str(credential, "id"),
		str(credential, "label"),
		str(credential, "provider"),
		str(credential, "credentialType"),
		maskSecret(str(credential, "apiKey")),
	}}
	return cmd.Output.Render(tableDef, rows, credential)
}
