package guardrails

import (
	"fmt"

	"github.com/revenium/revenium-cli/cmd"
)

// requireTeam refuses, before any HTTP request is issued, when no team has
// been resolved from the --team-id flag, the REVENIUM_TEAM_ID environment
// variable, or the config file's team-id key (D-27-10, D-31-26).
//
// Why this guard is needed at all in cmd/guardrails: the org-unit-group-preview
// operation's request schema (OrgUnitBudgetGroupPreviewRequest) declares BOTH
// parentOrgUnitId and teamId required, and teamId is a BODY property. The
// shared client injects teamId into the QUERY STRING (internal/api/client.go
// resolveURL), which does not populate a body — so an unresolved team here
// would send a body missing a required property and come back as a server-side
// error the operator has to decode. Every other command in this package leaves
// teamId entirely to the client, which is why this is the only file in
// cmd/guardrails that handles the team explicitly.
//
// Why the guard is local to cmd/guardrails and not in internal/api: Client's
// resolveURL appends teamId only when Client.TeamID is non-empty, and some
// endpoints legitimately require no team, so a blanket client-side refusal
// would break them. This closes the one instance in this package, not the
// class. internal/api/client.go is deliberately NOT modified by this phase.
//
// This is the THIRD copy in the repo (cmd/jobs/team.go, cmd/billing/team.go,
// now here) and it stays local. Extraction into a shared helper remains
// deferred: any shared helper would need a per-endpoint allowlist, because
// some endpoints must NOT be guarded. This phase's own session-attribution
// read is a live example — `revenium sessions attribution` declares teamId
// optional and deliberately ships with no guard (D-31-20), so a blanket
// helper applied there would refuse a call the server accepts. A third
// instance strengthens the deferred case without changing the answer.
//
// The message must stand alone. cmd/root.go sets SilenceUsage: true and
// SilenceErrors: true, so cobra prints no usage block after it, which is why
// the text names all three resolution sources itself rather than relying on
// cobra to show them.
func requireTeam() error {
	if cmd.APIClient == nil || cmd.APIClient.TeamID == "" {
		return fmt.Errorf(
			"no team resolved: this command requires a team.\n" +
				"Set one with --team-id <id>, the REVENIUM_TEAM_ID environment variable, " +
				"or `revenium config set team-id <id>`.")
	}
	return nil
}
