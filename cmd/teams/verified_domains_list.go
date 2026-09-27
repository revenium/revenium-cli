package teams

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newVerifiedDomainsListCmd returns the `list <team-id>` sub-command for
// verified-domains. It is a READ: it carries no Annotations{"mutating"} and no
// cmd.DryRun() gate, and it must not acquire either.
func newVerifiedDomainsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <team-id>",
		Short: "List the verified domains for a team's tenant",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List the tenant's verified domains
  revenium teams verified-domains list team-123

  # List as JSON
  revenium teams verified-domains list team-123 --json

  # Show only the domain column
  revenium teams verified-domains list team-123 --fields Domain`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/teams/%s/settings/verified-domains", url.PathEscape(args[0]))

			// The 200 is a PLAIN ARRAY of VerifiedDomainResource. It is not a
			// CollectionModel, so there is no collection wrapper to unwrap here
			// — and unwrapping one that is not present would silently render an
			// empty list. The sibling session-attribution read in this same
			// phase DOES unwrap its wrapper, because its response really is a
			// CollectionModel; the difference is in the two specs, not in taste.
			var domains []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &domains); err != nil {
				return err
			}

			// An empty collection is an ordinary case, not an error: a tenant
			// may simply have verified nothing yet. Empty-state branching per
			// cmd/jobs/list.go — a typed empty array under --json so a machine
			// consumer's parse still succeeds, a sentence otherwise.
			//
			// The sentence goes to the formatter's OWN writer, not to
			// c.OutOrStdout(): the formatter substitutes io.Discard in quiet
			// non-JSON mode (internal/output/output.go), so quiet suppression
			// is a property of the formatter rather than of this call site, and
			// a line written past the formatter escapes it (code review WR-03,
			// IN-06). This is the writer
			// cmd/guardrails/org_unit_group_preview.go already uses. The JSON
			// arm above is untouched — a document must survive --quiet.
			if len(domains) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(cmd.Output.Writer(), "No verified domains found for this team.")
				return nil
			}

			// Sort client-side, because this GET's spec states nothing about
			// row order. Making order a property of OUR output is the standing
			// rule (D-27-08) — the alternative is asserting a contract the spec
			// never states. This phase departs from that rule exactly once, for
			// the session-attribution read, and only because that GET's spec
			// DOES state its order as a contract.
			//
			// SliceStable, not Slice: rows comparing equal on `domain` keep the
			// server's relative order, so the output is deterministic across
			// runs rather than merely sorted.
			//
			// The decoded slice is sorted, not the rendered rows, so the raw
			// payload handed to --json reflects what the table shows.
			sort.SliceStable(domains, func(i, j int) bool {
				return str(domains[i], "domain") < str(domains[j], "domain")
			})

			return cmd.Output.Render(verifiedDomainsTableDef, verifiedDomainRows(domains), domains)
		},
	}
}
