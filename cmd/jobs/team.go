package jobs

import (
	"fmt"

	"github.com/revenium/revenium-cli/cmd"
)

// requireTeam refuses, before any HTTP request is issued, when no team has
// been resolved from the --team-id flag, the REVENIUM_TEAM_ID environment
// variable, or the config file's team-id key (D-27-10, SC5).
//
// Why this guard is local to cmd/jobs and not in internal/api: Client's
// resolveURL appends teamId only when Client.TeamID is non-empty, and some
// endpoints legitimately require no team, so a blanket client-side refusal
// would break them. This closes the four
// /v2/api/jobs/types/{type}/{economics,baselines} instances, not the class.
// internal/api/client.go is deliberately NOT modified by this phase.
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
