package teams

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// prHealthCmd is the parent pr-health subcommand under teams.
var prHealthCmd = &cobra.Command{
	Use:   "pr-health",
	Short: "Manage PR health thresholds for a team",
	Example: `  # View the effective PR health thresholds
  revenium teams pr-health get team-123

  # Update the thresholds
  revenium teams pr-health set team-123 --aging-days 14 --rotting-days 30`,
}

// initPrHealth registers pr-health subcommands. Called from teams.go init()
// to avoid file-ordering issues with Go's init() functions.
func initPrHealth() {
	prHealthCmd.AddCommand(newPrHealthGetCmd())
	prHealthCmd.AddCommand(newPrHealthSetCmd())
}

// prHealthTableDef defines the table layout for PR health settings output.
//
// This is a deliberate third copy of the `Setting | Value` definition already
// declared by prompt-capture and coding-assistant-filter. De-duplicating the
// three into one shared declaration is an existing deferred item and is
// intentionally not opened here (D-31 assumption-delta decision: add-alongside).
var prHealthTableDef = output.TableDef{
	Headers:      []string{"Setting", "Value"},
	StatusColumn: -1,
}

// renderPrHealthSettings renders PR health settings as a key-value table or JSON.
//
// Two details are load-bearing and copied verbatim from renderPromptSettings:
//   - `_links` is skipped, so HAL navigation noise never becomes a settings row.
//   - rows are sorted by key, because Go map iteration order is non-deterministic
//     and without the sort the rendered row order is random per run.
//
// The third argument to Render is the untouched server payload: that is what
// --json emits, which is what keeps the table lossless. Routing through
// cmd.Output.Render (rather than RenderTable directly) is also what makes
// --fields actually narrow the columns (D-31-33 / T-30-11).
func renderPrHealthSettings(settings map[string]interface{}) error {
	var rows [][]string
	for key, val := range settings {
		if key == "_links" {
			continue
		}
		rows = append(rows, []string{key, fmt.Sprint(val)})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})
	return cmd.Output.Render(prHealthTableDef, rows, settings)
}
