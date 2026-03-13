// Package meter implements the metering event submission commands for the Revenium CLI.
package meter

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent meter command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "meter",
	Short: "Submit metering events",
	Example: `  # Meter a generic event
  revenium meter event --transaction-id txn-123 --payload '{"apiCalls": 100}'

  # Meter an AI completion
  revenium meter completion --model gpt-4 --provider openai --input-tokens 500 --output-tokens 200 --total-tokens 700

  # Meter an API request
  revenium meter api-request --transaction-id txn-456 --method POST --resource /api/users`,
	PersistentPreRunE: func(c *cobra.Command, args []string) error {
		// Run the root PersistentPreRunE first (config/API client init).
		if root := c.Root(); root != nil && root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(c, args); err != nil {
				return err
			}
		}
		// Metering endpoints use a different base path (/meter) than the
		// management API (/profitstream). Swap the base URL after init.
		if cmd.APIClient != nil {
			cmd.APIClient.BaseURL = cmd.APIClient.MeterBaseURL()
		}
		return nil
	},
}

func init() {
	Cmd.AddCommand(newEventCmd())
	Cmd.AddCommand(newAPIRequestCmd())
	Cmd.AddCommand(newAPIResponseCmd())
	Cmd.AddCommand(newCompletionCmd())
	Cmd.AddCommand(newImageCmd())
	Cmd.AddCommand(newAudioCmd())
	Cmd.AddCommand(newVideoCmd())
	Cmd.AddCommand(newToolEventCmd())
}

// responseDef defines the table layout for metering response output.
var responseDef = output.TableDef{
	Headers:      []string{"ID", "Type", "Label", "Created"},
	StatusColumn: -1,
}

// renderResponse renders a metering response as a single-row table or JSON.
func renderResponse(result map[string]interface{}) error {
	rows := [][]string{{
		str(result, "id"),
		str(result, "resourceType"),
		str(result, "label"),
		str(result, "created"),
	}}
	return cmd.Output.Render(responseDef, rows, result)
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
