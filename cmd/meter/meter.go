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
}

func init() {
	// Assigned here rather than inside the composite literal above: naming Cmd
	// within its own initializer is the Go initialization cycle
	// ("initialization cycle: Cmd refers to itself").
	//
	// The root PersistentPreRunE has to run first because it is what
	// initializes config and cmd.APIClient; skipping it would leave the
	// shipped `revenium meter ...` commands calling into a nil client — and
	// the base-URL swap below with nothing to swap.
	//
	// Why the guard compares root against Cmd and not against c: cobra walks
	// up from the executed command to the nearest ancestor carrying a
	// PersistentPreRunE, then invokes that hook passing the executed LEAF as
	// c. Running Cmd unattached with `event --transaction-id X` therefore
	// enters here with c == the event command and root == Cmd, so the leaf
	// form would never trip and this closure would call itself until the
	// process dies with `fatal error: stack overflow`. Comparing against the
	// hook's OWNER is what actually detects "Cmd is its own root", i.e. that
	// the root hook we would delegate to IS this very closure.
	//
	// cmd/metrics/dimensions.go carries the leaf-command variant of this same
	// guard (comparing root against c); that form is correct there precisely
	// because dimensions is a leaf, so there c IS the hook's owner. The
	// divergence is deliberate, not drift.
	Cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		// Run the root PersistentPreRunE first (config/API client init) —
		// but only when a real root sits above Cmd, as it does under
		// main.go's rootCmd.
		if root := c.Root(); root != nil && root != Cmd && root.PersistentPreRunE != nil {
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
	}

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
