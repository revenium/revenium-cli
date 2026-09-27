package teams

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// attributionIdentityPolicyCmd is the parent attribution-identity-policy
// subcommand under teams.
//
// Its Example writes the set verb's argument as a `<policy>` placeholder rather
// than spelling a policy value out. This file is on the READ path, whose
// acceptance gate strips comments and then requires ZERO occurrences of the
// VERIFIED_DOMAIN_ONLY literal in it — see renderAttributionIdentityPolicy
// below for why that gate exists. The set command's own file names both legal
// values in its --policy usage string and in its own Example, which is where an
// operator looks for them.
var attributionIdentityPolicyCmd = &cobra.Command{
	Use:   "attribution-identity-policy",
	Short: "Manage the coding-assistant subscriber identity policy for a team",
	Example: `  # View the stored subscriber identity policy
  revenium teams attribution-identity-policy get team-123

  # Update the stored subscriber identity policy
  revenium teams attribution-identity-policy set team-123 --policy <policy>`,
}

// initAttributionIdentityPolicy registers attribution-identity-policy
// subcommands. Called from teams.go init() to avoid file-ordering issues with
// Go's init() functions.
func initAttributionIdentityPolicy() {
	attributionIdentityPolicyCmd.AddCommand(newAttributionIdentityPolicyGetCmd())
	attributionIdentityPolicyCmd.AddCommand(newAttributionIdentityPolicySetCmd())
}

// attributionIdentityPolicyTableDef defines the table layout for the
// attribution identity policy output.
//
// This is a deliberate fourth copy of the `Setting | Value` definition already
// declared by prompt-capture, coding-assistant-filter and pr-health.
// De-duplicating the four into one shared declaration is an existing deferred
// item and is intentionally not opened here (31-02 deliberate_non_actions).
var attributionIdentityPolicyTableDef = output.TableDef{
	Headers:      []string{"Setting", "Value"},
	StatusColumn: -1,
}

// renderAttributionIdentityPolicy renders the attribution identity policy as a
// key-value table or JSON.
//
// Three details are load-bearing and copied from renderPrHealthSettings:
//   - `_links` is skipped, so HAL navigation noise never becomes a settings row.
//   - rows are sorted by key, because Go map iteration order is non-deterministic
//     and without the sort the rendered row order is random per run.
//   - the third argument to Render is the untouched server payload: that is what
//     --json emits, which is what keeps the table lossless. Routing through
//     cmd.Output.Render (rather than RenderTable directly) is also what makes
//     --fields actually narrow the columns (D-31-33 / T-30-11).
//
// The fourth detail is what this renderer deliberately does NOT do, and it is
// the reason this whole read path is gated against a hardcoded policy literal
// (D-31-02, T-31-09):
//
// ROADMAP SC2 asks that an unset policy be reported as the strict
// verified-domain value rather than as blank. The SERVER already does that. The
// GET operation description states that an unset policy is returned as the
// strict policy, and `policy` is `required` in AttributionIdentityPolicyResource.
// SC2 is therefore satisfied by rendering faithfully — there is nothing left for
// the client to substitute.
//
// Adding a client-side blank-to-strict fallback would be the same invention as a
// client-side defaults table (D-22-04, cmd/skills/skills.go), and worse: it would
// mask a genuine contract break — a server that starts returning a blank policy —
// behind a value the CLI made up, at security stakes. Do not "help" by adding it.
// TestAttributionIdentityPolicyGetRendersBlankVerbatim asserts its ABSENCE.
func renderAttributionIdentityPolicy(settings map[string]interface{}) error {
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
	return cmd.Output.Render(attributionIdentityPolicyTableDef, rows, settings)
}
