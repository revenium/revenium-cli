package billing

import (
	"fmt"

	"github.com/revenium/revenium-cli/cmd"
)

// requireTeam refuses, before any HTTP request is issued, when no team has been
// resolved from the --team-id flag, the REVENIUM_TEAM_ID environment variable,
// or the config file's team-id key (D-30-05).
//
// Why the guard is needed at all: the seats operation declares teamId as a
// required query parameter, but the shared client appends it in resolveURL
// (internal/api/client.go:72-91) only when Client.TeamID is non-empty. Without
// this guard an unresolved team produces a request that reaches the server
// missing a required parameter and comes back as a server-side error the
// operator has to decode.
//
// Why it is local to cmd/billing and not in internal/api: some endpoints
// legitimately require no team, so a blanket client-side refusal would break
// them. This closes the instances in this package, not the class.
// internal/api/client.go is deliberately NOT modified by this phase. The
// package-local duplication across cmd/* is the established convention here
// (billing.go:41-43 states it for str()).
//
// D-30-05: this command registers no team flag of its own. The global
// --team-id override already exists (README.md:81) and a second one would
// collide.
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
