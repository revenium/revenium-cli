package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// validatePrHealthOrder refuses a threshold pair the server would reject,
// before any request is built.
//
// Source: the PrHealthSettingsResource property descriptions and the PUT
// operation description — agingDays "Must be lower than" rottingDays, and
// rottingDays "Must be higher than" agingDays. Equality satisfies neither, so
// the refusal is >=, not >. If upstream ever loosens that constraint, this
// check reads as stale rather than as a mystery.
//
// Deliberately NOT enforced here: any lower bound, upper bound or
// non-negativity rule. The schema declares no such bound on either property.
// Inventing a floor client-side is the same invention as inventing a default,
// in the opposite direction — it would refuse a call the server accepts. Zero
// and negative values are therefore passed through and the server owns that
// rule.
func validatePrHealthOrder(aging, rotting int) error {
	if aging < rotting {
		return nil
	}
	return fmt.Errorf("--aging-days (%d) must be less than --rotting-days (%d)", aging, rotting)
}

// newPrHealthSetCmd returns the `set <team-id>` sub-command for PR health
// settings. It PUTs both thresholds in one request.
//
// There is deliberately no read-then-merge here (D-31-03), and that is a
// considered departure from the rule that governs `jobs economics set`. There,
// the body is a large multi-field contract and a partial write would destroy
// fields the operator never mentioned, so the current state must be read first.
// Here the body is two mutually constrained integers that the resource declares
// both required: a one-flag call cannot form a legal body at all. Requiring both
// flags makes the ordering guard TOTAL — there is no call shape it cannot
// evaluate — and because no read sits between the check and the write, there is
// no client-side window in which the value checked and the value written could
// diverge.
func newPrHealthSetCmd() *cobra.Command {
	var (
		agingDays   int
		rottingDays int
	)

	c := &cobra.Command{
		Use:   "set <team-id>",
		Short: "Update PR health thresholds for a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Set both thresholds
  revenium teams pr-health set team-123 --aging-days 14 --rotting-days 30

  # Preview the request without sending it
  revenium teams pr-health set team-123 --aging-days 14 --rotting-days 30 --dry-run`,
		Annotations: map[string]string{"mutating": "true"},
		// The ordering guard runs here, not inside RunE, so it fires at zero
		// HTTP cost and the refusal is provable by a zero request counter
		// rather than by the mere presence of an error (T-31-04).
		//
		// It is SKIPPED when either threshold flag was not passed, because
		// cobra validates required flags AFTER PreRunE (cobra command.go:
		// PreRunE at :969, ValidateRequiredFlags at :977). Comparing unset
		// zero values here replaced cobra's precise "required flag(s) ... not
		// set" with a vaguer arithmetic message that told the operator two
		// values they never supplied were out of order, and left both
		// MarkFlagRequired calls below as unreachable dead code (code review
		// WR-01). The same shape is used in
		// cmd/guardrails/org_unit_group_preview.go.
		//
		// The zero-HTTP guarantee ROADMAP SC1 names is unaffected either way —
		// cobra's validation also refuses before any request is built. Only
		// the message an operator reads changes.
		PreRunE: func(c *cobra.Command, args []string) error {
			agingPassed := c.Flags().Changed("aging-days")
			rottingPassed := c.Flags().Changed("rotting-days")
			if !agingPassed || !rottingPassed {
				return nil
			}
			return validatePrHealthOrder(agingDays, rottingDays)
		},
		RunE: func(c *cobra.Command, args []string) error {
			// Both values are sent unconditionally: the resource declares both
			// properties required, so the conditional per-field gating used for
			// optional fields elsewhere does not apply.
			body := map[string]interface{}{
				"agingDays":   agingDays,
				"rottingDays": rottingDays,
			}

			// Built before the dry-run branch so the previewed path and the sent
			// path are literally the same string.
			path := fmt.Sprintf("/v2/api/teams/%s/settings/pr-health", url.PathEscape(args[0]))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "PR health settings", path, body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}
			return renderPrHealthSettings(result)
		},
	}

	c.Flags().IntVar(&agingDays, "aging-days", 0, "Days without activity before an open PR counts as aging; must be lower than --rotting-days")
	c.Flags().IntVar(&rottingDays, "rotting-days", 0, "Days without activity before an open PR counts as rotting; must be higher than --aging-days")

	// Both thresholds are enforced at the Cobra layer so an under-specified
	// invocation fails BEFORE any HTTP round-trip. These two calls are live
	// code only because PreRunE above yields when a flag was not passed;
	// evaluating unset flags there would swallow the message they produce.
	_ = c.MarkFlagRequired("aging-days")
	_ = c.MarkFlagRequired("rotting-days")

	return c
}
