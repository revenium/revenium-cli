// Package sessions implements the read side of coding-assistant session
// attribution: `revenium sessions attribution <session-id>`, which lists the
// ticket-attribution intervals the platform has recorded for a session
// (SESS-01).
//
// WHY THIS IS ITS OWN TOP-LEVEL PACKAGE RATHER THAN A VERB ON AN EXISTING
// GROUP (D-31-15).
//
// The cheaper move is a read-only verb on a group that already exists. It was
// taken up and rejected, group by group, because there is no group a session
// belongs to:
//
//   - teams — a session is not a team resource. `teamId` is an OPTIONAL query
//     filter on the attribution GET, not the session's parent. Filing it under
//     `teams` would assert a containment the API does not declare.
//   - squads — squads are execution groups (a squad runs a job). A session is
//     a different noun: one coding-assistant conversation, attributed over
//     time to one or more tickets.
//   - metrics — this files a RESOURCE under a QUERY surface purely because of
//     which host happens to serve it. D-29-01 rejected exactly this move for
//     `roi-summary`, and the reasoning holds here with the host clause
//     removed. `metrics` also owns `--from`/`--to` persistent flags that do
//     not map to anything on this operation.
//   - billing / skills — coding-assistant-adjacent, so tempting. But a session
//     is neither a billing concept (it carries no charge) nor a skills concept
//     (it is not a skill's usage rollup).
//
// cmd/skills is the shipped precedent for precisely this situation: a new
// top-level resource package rather than nesting under a group whose flags do
// not map. cmd/squads is the same shape one generation earlier.
//
// The cost side, stated honestly: a package costs exactly one
// RegisterCommand line in main.go plus its import. A wrong home costs
// permanently — a released command path is a published contract, so moving
// `revenium sessions attribution` later means a deprecation notice in the
// register BILL-07 established, not a rename.
//
// THE NO-HOST-SWAP PIN (D-31-32).
//
// The attribution GET lives on the PLATFORM API document, reached with
// x-api-key — not metering, not analytics. This package therefore uses
// cmd.APIClient exactly as configured and deliberately makes NO base-URL
// change: no meter base URL, no analytics base URL, no bearer auth, and no
// PersistentPreRunE base-URL mutation. Adding one would silently retarget a
// read command at a surface that does not serve it, and it would additionally
// make tools/coverage-audit/extract.go's surfaceRules need an edit this phase
// asserts it does not need.
package sessions

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the parent sessions command, exported for registration in main.go.
// It carries no Annotations{"mutating": ...} because its only subcommand is
// read-only: the write side of session attribution
// (POST /v2/api/sessions/{sessionId}/attribution) is a recorded deliberate
// skip and does not live here.
var Cmd = &cobra.Command{
	Use:   "sessions",
	Short: "Inspect coding-assistant sessions",
	Example: `  # List a session's ticket attribution intervals, current interval first
  revenium sessions attribution sess-8f2a1c

  # Emit the intervals as a JSON array, one object per interval
  revenium sessions attribution sess-8f2a1c --json`,
}

func init() {
	// One leaf subcommand, no sub-parent, so no initX() indirection is
	// needed. That convention exists to dodge file-ordering between a parent
	// declared in one file and children registered in another; there is only
	// one child here and it is added directly.
	Cmd.AddCommand(newAttributionCmd())
}

// str safely extracts a string value from a map, returning "" for missing or
// nil keys. A package-local copy of the helper cmd/teams, cmd/skills,
// cmd/squads and cmd/metrics each carry: package sessions cannot import one
// from cmd/teams, and a shared home for it is a known, wanted, and
// deliberately-not-opened-here consolidation.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
