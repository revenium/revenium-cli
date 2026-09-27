package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newVerifiedDomainsAddCmd returns the `add <team-id> <domain>` sub-command for
// verified-domains.
//
// D-31-13 — there are deliberately NO --source and --join-policy flags.
// VerifiedDomainRequest declares a single property, `domain`. Flags for the
// other two columns would build a request the server ignores while telling the
// operator they chose something they did not. What the platform assigns is
// surfaced on the way back out instead, by rendering the PUT's own response.
func newVerifiedDomainsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <team-id> <domain>",
		Short: "Verify a domain for a team's tenant",
		Example: `  # Verify a domain for the tenant
  revenium teams verified-domains add team-123 acme.example

  # Preview the request without sending it
  revenium teams verified-domains add team-123 acme.example --dry-run`,
		// cmd.ValidResourceID is a SAFETY validator, applied here to both
		// positional arguments. It rejects empty values, control characters,
		// `?`, `&`, `#`, `%` and `../`. It is NOT a hostname or RFC format
		// check: it accepts `acme.example` and it accepts `40`.
		//
		// Do not describe it as a format check and do not add one.
		// VerifiedDomainRequest.domain declares no `pattern` and no `format`,
		// so a client-side hostname regex would refuse values the server
		// accepts.
		Args:        cobra.MatchAll(cobra.ExactArgs(2), cmd.ValidResourceID),
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			// D-31-12 — the operator's string, verbatim.
			//
			// VerifiedDomainResource.domain is documented as a lowercase
			// verified domain, but that is a statement about what the server
			// STORES. VerifiedDomainRequest.domain is an RFC hostname to
			// verify, with no pattern and no format. Lowercasing or trimming
			// here is a normalisation the server may or may not perform
			// identically — and on the sibling `remove`, where the domain is
			// matched against stored rows, a transform that disagrees with the
			// server's is a delete that silently misses while reporting
			// success. Send what the operator typed and let the server's answer
			// be the answer.
			//
			// This leaves one NAMED LIVE CHECK open, deliberately: whether
			// removal matches case-insensitively against a stored lowercase row
			// is one request against a real host, and it belongs on the UAT list
			// rather than in a client-side guess. A stub would only confirm
			// whatever we told it to match.
			body := map[string]interface{}{"domain": args[1]}

			path := fmt.Sprintf("/v2/api/teams/%s/settings/verified-domains", url.PathEscape(args[0]))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "add", "verified domain", path, body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}

			// D-31-13 — render all three properties of the mapping that was
			// just created, through the same TableDef `list` uses. The PUT's
			// description states that new mappings are given join policy
			// REQUEST and source ADMIN; both come back in the response, so
			// rendering them tells the operator what the platform attached to
			// the domain they verified. Dropping them would leave an operator
			// to discover the join policy later.
			return cmd.Output.Render(verifiedDomainsTableDef, verifiedDomainRows([]map[string]interface{}{result}), result)
		},
	}

	return c
}
