package teams

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// validAttributionIdentityPolicies is the exact two-value enum declared by
// AttributionIdentityPolicyResource.policy in the 2.20.0-SNAPSHOT dev platform
// document. It is the only source for this list.
//
// The comparison against it is EXACT — no case folding and no trimming. The
// schema declares these two values in one casing, and normalising the input
// first would accept something the server never promised to take: the CLI would
// be inventing an accepted input, the mirror image of inventing a default value.
//
// If upstream ever widens the enum, this check reads as stale rather than as a
// mystery — that is the whole point of naming the source here (D-31-08, the
// validateDimensionName precedent in cmd/metrics/dimensions.go).
var validAttributionIdentityPolicies = []string{
	"VERIFIED_DOMAIN_ONLY",
	"ALLOW_SELF_ASSERTED_UNVERIFIED",
}

// validateAttributionIdentityPolicy refuses anything outside the schema enum
// before any request is built. Wired as the command's PreRunE, so the refusal
// costs zero HTTP and is provable by a zero request counter rather than by the
// mere presence of an error (T-31-06).
func validateAttributionIdentityPolicy(policy string) error {
	for _, v := range validAttributionIdentityPolicies {
		if policy == v {
			return nil
		}
	}
	return fmt.Errorf("policy %q is not valid (expected one of: %s)",
		policy, strings.Join(validAttributionIdentityPolicies, ", "))
}

// attributionIdentityPolicyNotEnforcedNotice is D-31-06's text.
//
// Four facts a future reader needs, the first three copied in form from the
// shipped precedent at cmd/billing/claude_code_contributions.go:44-57:
//
//  1. It goes to the cobra command's error writer, never through cmd.Output —
//     the formatter writes to stdout (internal/output/output.go), so a notice
//     there would break every --json consumer's parse.
//  2. It is emitted BEFORE the body is built and before the dry-run gate, so it
//     reaches the operator on a dry run and on a failed write alike, not only on
//     the runs that happen to succeed.
//  3. The substantive fact: an operator who runs this command with the strict
//     value, gets a 200 and sees the strict value echoed back will reasonably
//     conclude they have closed a hole. They have not. The stored value is
//     recorded intent that the platform currently ignores. That is a
//     wrong-but-plausible outcome at security stakes, which is why the notice
//     exists at all.
//  4. Its source, and its expiry: this text is sourced from the `policy`
//     property description in the 2.20.0-SNAPSHOT dev schema, which describes a
//     TEMPORARY platform state ("while the verified-domain gate is
//     deactivated"). When that gate is activated the paragraph changes and this
//     notice becomes WRONG. The next drift audit that sees the paragraph change
//     should delete this constant, its unconditional print in RunE and the test
//     that pins it — that is the whole delete, and naming it here is what makes
//     it a delete rather than an investigation (D-31-09).
//
// It is deliberately UNCONDITIONAL — see the print site in RunE for why
// conditioning it on the value being set is the failure mode (D-31-07).
const attributionIdentityPolicyNotEnforcedNotice = "Note: this policy is recorded intent and is not currently enforced. " +
	"While the platform's verified-domain gate is deactivated, attribution resolves every team to " +
	"ALLOW_SELF_ASSERTED_UNVERIFIED and accepts structurally valid identities from any domain; " +
	"structurally invalid addresses are rejected either way."

// newAttributionIdentityPolicySetCmd returns the `set <team-id>` sub-command for
// the team's coding-assistant subscriber identity policy. It PUTs the single
// required property in one request and re-renders the PUT's own 200 response
// through the shared renderer (D-31-05).
//
// There is deliberately no read-then-merge: the resource declares exactly one
// property and declares it required, so there is no field a partial write could
// destroy and nothing to merge with.
func newAttributionIdentityPolicySetCmd() *cobra.Command {
	var policy string

	c := &cobra.Command{
		Use:   "set <team-id>",
		Short: "Update the coding-assistant subscriber identity policy for a team",
		Long: `Update the stored coding-assistant subscriber identity policy for a team.

The stored value is recorded intent. While the platform's verified-domain gate is
deactivated, attribution resolves every team to ALLOW_SELF_ASSERTED_UNVERIFIED
regardless of what is stored here, so a successful write does not by itself make
a team stricter. The command prints that fact on stderr before every write.`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Record the strict policy
  revenium teams attribution-identity-policy set team-123 --policy VERIFIED_DOMAIN_ONLY

  # Record the permissive policy
  revenium teams attribution-identity-policy set team-123 --policy ALLOW_SELF_ASSERTED_UNVERIFIED

  # Preview the request without sending it
  revenium teams attribution-identity-policy set team-123 --policy VERIFIED_DOMAIN_ONLY --dry-run`,
		Annotations: map[string]string{"mutating": "true"},
		// The enum guard runs here, not inside RunE, so it fires at zero HTTP
		// cost and the refusal is provable by a zero request counter.
		//
		// It is SKIPPED when --policy was not passed at all, because cobra
		// validates required flags AFTER PreRunE (cobra command.go: PreRunE at
		// :969, ValidateRequiredFlags at :977). Validating the unset empty
		// string here replaced cobra's precise "required flag(s) ... not set"
		// with an enum message about a value the operator never supplied, and
		// left MarkFlagRequired below unreachable (code review IN-01, WR-01 on
		// pr_health_set.go, and the same shape in
		// cmd/guardrails/org_unit_group_preview.go).
		//
		// An explicitly EMPTY --policy is still this guard's case, and must be:
		// cobra counts such a flag as supplied, so its own validation would let
		// an empty body through to the server. The zero-HTTP guarantee holds on
		// both paths — only the message an operator reads differs.
		PreRunE: func(c *cobra.Command, args []string) error {
			if !c.Flags().Changed("policy") {
				return nil
			}
			return validateAttributionIdentityPolicy(policy)
		},
		RunE: func(c *cobra.Command, args []string) error {
			// FIRST statement, before the body is built and before the dry-run
			// gate, so the operator sees it on a dry run and on a failed write
			// alike. See the constant above for the full reasoning and for the
			// schema paragraph that will one day make this line deletable.
			//
			// UNCONDITIONAL, deliberately (D-31-07). Warning only on the strict
			// value — the one the platform currently ignores — and staying quiet
			// on the permissive one is tempting and wrong: it encodes a
			// client-side belief about which server-side gate is live right now,
			// and the moment that gate is activated the conditional is silently
			// wrong in the other direction. The notice states the situation; it
			// does not predict which value is affected by it.
			fmt.Fprintln(c.ErrOrStderr(), attributionIdentityPolicyNotEnforcedNotice)

			// One key, unconditionally: the resource declares exactly one
			// property and declares it required, so the conditional per-field
			// gating used for optional fields elsewhere does not apply.
			body := map[string]interface{}{"policy": policy}

			// Built before the dry-run branch so the previewed path and the sent
			// path are literally the same string.
			path := fmt.Sprintf("/v2/api/teams/%s/settings/attribution-identity-policy", url.PathEscape(args[0]))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "attribution identity policy", path, body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}
			return renderAttributionIdentityPolicy(result)
		},
	}

	c.Flags().StringVar(&policy, "policy", "", "Subscriber identity policy to store (VERIFIED_DOMAIN_ONLY or ALLOW_SELF_ASSERTED_UNVERIFIED)")

	// The flag is enforced at the Cobra layer: there is no legal empty body for
	// a resource whose only property is required. Since 31-08 this is not a
	// backstop but the mechanism that actually reports an omitted --policy —
	// PreRunE above yields to it rather than answering first with the enum.
	_ = c.MarkFlagRequired("policy")

	return c
}
