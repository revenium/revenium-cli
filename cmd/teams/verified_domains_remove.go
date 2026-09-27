package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// newVerifiedDomainsRemoveCmd returns the `remove <team-id> <domain>`
// sub-command for verified-domains.
//
// The RunE ordering below is copied from the shipped team delete
// (cmd/teams/delete.go) and is load-bearing: the dry-run gate is checked BEFORE
// the confirmation prompt, so a dry run never asks an operator to confirm a
// deletion it is not going to perform.
func newVerifiedDomainsRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "remove <team-id> <domain>",
		Short: "Remove a verified domain from a team's tenant",
		Example: `  # Remove a verified domain (with confirmation)
  revenium teams verified-domains remove team-123 acme.example

  # Remove without confirmation
  revenium teams verified-domains remove team-123 acme.example --yes

  # Preview the request without sending it
  revenium teams verified-domains remove team-123 acme.example --dry-run`,
		// cmd.ValidResourceID is a SAFETY validator applied to both positional
		// arguments: it rejects empty values, control characters, `?`, `&`,
		// `#`, `%` and `../`. It is NOT a hostname or RFC format check — it
		// accepts `acme.example` and it accepts `40`. Do not add a format check
		// here; the request schema declares none.
		Args:        cobra.MatchAll(cobra.ExactArgs(2), cmd.ValidResourceID),
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			// QueryEscape for the domain, PathEscape for the team id. The two
			// escape different character sets, and using the wrong one on a
			// query value produces a request that reaches the server carrying a
			// different string than the operator typed.
			//
			// D-31-12, with the sharper stake than on `add`: the `domain` query
			// parameter is REQUIRED and is matched against stored rows. Any
			// client-side transform — lowercasing, trimming, normalising — that
			// disagrees with the server's is a delete that removes NOTHING
			// while the command reports success. Send what the operator typed;
			// let a 404 be honest.
			path := fmt.Sprintf("/v2/api/teams/%s/settings/verified-domains?domain=%s",
				url.PathEscape(args[0]), url.QueryEscape(args[1]))

			// BEFORE the confirmation prompt, deliberately: a dry run must not
			// prompt. This is the ordering cmd/teams/delete.go establishes.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "remove", "verified domain", path, nil)
			}

			// --yes is a PERSISTENT ROOT flag, not one declared on this
			// command; read it here rather than declaring a local duplicate.
			yes, _ := c.Flags().GetBool("yes")

			// The DOMAIN is the id argument, not a composite string naming the
			// team. That argument is what the prompt echoes, and it should echo
			// the thing being deleted — the team is already on the command
			// line. resource.ConfirmDelete is reused unchanged (D-31-11).
			ok, err := resource.ConfirmDelete("verified domain", args[1], yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			// The error is returned unchanged. A 404 from a domain that was
			// already removed, or whose case did not match what the server
			// stored, is the server's honest answer — relayed rather than
			// reinterpreted into a success or a guess.
			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
				return err
			}

			// WR-04. A machine consumer parses this stream, and a 204 carries
			// no body to relay — so the success envelope is the CLI's OWN
			// document, chosen here and therefore a shape that must stay
			// stable. It names the domain and the team so a caller confirms
			// WHICH row was removed rather than inferring it from the
			// arguments it happened to pass.
			//
			// The branch sits AFTER the DELETE deliberately: a failed delete
			// returns above and never emits a success document. It also sits
			// ahead of the prose print, which is the whole point — the two
			// must never share the stream. This is the discipline
			// cmd/sessions/attribution.go already applies to its hint.
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(map[string]interface{}{
					"removed": true,
					"domain":  args[1],
					"teamId":  args[0],
				})
			}

			// Written to the formatter's OWN writer, not to c.OutOrStdout():
			// the formatter substitutes io.Discard in quiet non-JSON mode
			// (internal/output/output.go), so quiet suppression becomes a
			// property of the formatter rather than of this call site. That is
			// what makes the explicit IsQuiet() guard that used to wrap this
			// print redundant — keeping both would state the suppression twice
			// and let the two statements drift (code review IN-06). This is
			// the writer cmd/guardrails/org_unit_group_preview.go already uses.
			fmt.Fprintf(cmd.Output.Writer(), "Removed verified domain %s from team %s.\n", args[1], args[0])
			return nil
		},
	}

	return c
}
