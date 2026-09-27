package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newAttributionIdentityPolicyGetCmd returns the `get <team-id>` sub-command
// for the team's coding-assistant subscriber identity policy.
//
// It carries no Annotations{"mutating": ...} — it is a read.
//
// It also prints NO stderr notice, deliberately (D-31-06). A read reporting a
// stored value makes no claim about enforcement, and a stderr line on every read
// trains operators to ignore stderr — at which point the notice that matters,
// the one on `set` immediately before the write, is the one they miss. The
// enforcement fact lives in this command's Long help in prose instead.
// TestAttributionIdentityPolicyGetPrintsNoNotice asserts the error writer stays
// empty.
func newAttributionIdentityPolicyGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <team-id>",
		Short: "View the coding-assistant subscriber identity policy for a team",
		Long: `View the stored coding-assistant subscriber identity policy for a team.

The value reported here is stored intent, not a description of what the platform
is enforcing right now. While the platform's verified-domain gate is deactivated,
attribution resolves every team to ALLOW_SELF_ASSERTED_UNVERIFIED and accepts
structurally valid identities from any domain; structurally invalid addresses are
rejected either way. A team whose stored policy requires verified domains
therefore still accepts self-asserted identities today.

The command renders exactly what the server sent. An unset policy is already
returned by the API as the strict policy, so the CLI substitutes nothing: a blank
value would be a real contract break and is shown as one rather than masked.`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View the stored subscriber identity policy
  revenium teams attribution-identity-policy get team-123

  # View as JSON
  revenium teams attribution-identity-policy get team-123 --json

  # Show only the values
  revenium teams attribution-identity-policy get team-123 --fields Value`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/teams/%s/settings/attribution-identity-policy", url.PathEscape(args[0]))

			var settings map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &settings); err != nil {
				return err
			}

			// Nothing sits between the decode and the render, and that gap is
			// the load-bearing part of this command (D-31-02, T-31-09).
			//
			// ROADMAP SC2 asks that an unset policy be reported as the strict
			// verified-domain value rather than as blank. The SERVER already
			// does that: the GET operation description says an unset policy is
			// returned as the strict policy, and `policy` is `required` in
			// AttributionIdentityPolicyResource. SC2 is satisfied by rendering
			// faithfully.
			//
			// A client-side blank-to-strict substitution here would be D-22-04's
			// recorded anti-pattern (hardcoding a prose-documented server default
			// client-side, see cmd/skills/skills.go) and worse: it would print
			// VERIFIED_DOMAIN_ONLY where the server sent nothing, masking a real
			// contract break behind a value the CLI invented — at security
			// stakes, since the value being invented is the strict one. Do not
			// add it. TestAttributionIdentityPolicyGetRendersBlankVerbatim
			// asserts it is absent.
			return renderAttributionIdentityPolicy(settings)
		},
	}
}
