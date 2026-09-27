package guardrails

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/revenium/revenium-cli/internal/validate"
)

// orgUnitGroupPreviewPath is the single path this command ever calls. It is a
// package-level constant so the test that proves nothing else is called can
// name the same string the command sends.
const orgUnitGroupPreviewPath = "/v2/api/ai/cost-controls/org-unit-group-preview"

// newOrgUnitGroupPreviewCmd builds `revenium guardrails org-unit-group-preview`
// (GRDR-08) — a POST whose method lies. It asks the platform how many DIRECT
// sub-teams of an org unit a groupBy=ORG_UNIT budget rule would independently
// cap, and it creates nothing.
//
// READ THIS BEFORE "FIXING" THE MISSING MUTATION MARKER (D-31-23).
//
// This command deliberately carries NO Annotations{"mutating":"true"}, NO
// cmd.DryRun() gate and NO confirmation prompt. Every other POST in
// cmd/guardrails (budget_rules_create.go, budget_rules_update.go,
// budget_rules_delete.go) carries the mutating marker, so a reviewer or a
// later agent scanning this package WILL see the asymmetry and try to correct
// it. The correction would be wrong.
//
// The source is the operation description itself: "read-only preview for the
// 'group by Department' UI flow — returns the direct sub-teams of
// parentOrgUnitId and their count, for the 'this will independently cap N
// teams' confirmation copy. Creates nothing: the actual budget is a single
// POST /v2/api/ai/cost-controls write with groupBy: 'ORG_UNIT'."
//
// cmd/skills/skills.go records the same "carries no mutation surface because it
// is read-only" reasoning for a GET. This is the harder sibling of that case:
// there the HTTP method is the evidence, here the method is not evidence at
// all, so the evidence has to be this comment plus
// TestOrgUnitGroupPreviewSendsOneRequestAndCreatesNothing, which asserts the
// stub saw exactly one request, to this path, and none of any other.
//
// Why no dry-run gate: a dry run of a read prints "would preview", which is
// noise — there is no write to withhold. Why no confirmation prompt: a prompt
// would teach an operator that this command changes something, which is
// precisely the mental model GRDR-08 exists to avoid.
func newOrgUnitGroupPreviewCmd() *cobra.Command {
	var parentOrgUnitID string

	c := &cobra.Command{
		Use:   "org-unit-group-preview",
		Short: "Preview how many sub-teams a department-scoped budget would cap (creates nothing)",
		Long: `Ask the platform how many DIRECT sub-teams of an org unit a department-scoped
budget rule (groupBy ORG_UNIT) would independently cap, and which sub-teams they are.

This command creates nothing. It is the confirmation figure you want BEFORE you
write such a rule with 'revenium guardrails budget-rules create --group-by ORG_UNIT';
the rule itself is a separate, explicitly mutating command.

The endpoint requires both the org-unit-budgets-enabled and the
org-unit-attribution-enabled feature flags to be enabled for the team. Without
them the platform answers HTTP 422 and no preview is produced.`,
		Args: cobra.NoArgs,
		Example: `  # How many sub-teams would a department-scoped budget cap under org unit 40?
  revenium guardrails org-unit-group-preview --parent-org-unit-id 40`,
		// The safety check runs here rather than in RunE so it costs zero HTTP
		// and fires before any body is assembled. It is skipped when the flag
		// was not passed at all, because cobra validates required flags AFTER
		// PreRunE (cobra command.go: PreRunE at :999, ValidateRequiredFlags at
		// :1007) — validating an unset flag here would replace cobra's precise
		// "required flag(s) not set" message with a vaguer empty-value one.
		PreRunE: func(c *cobra.Command, args []string) error {
			if !c.Flags().Changed("parent-org-unit-id") {
				return nil
			}
			return validate.ResourceID(parentOrgUnitID)
		},
		RunE: func(c *cobra.Command, args []string) error {
			// The team must be resolved BEFORE the body is built, because the
			// body carries it. See requireTeam() in team.go for why the guard
			// is local to this package.
			if err := requireTeam(); err != nil {
				return err
			}

			// teamId is written into the BODY explicitly here, which no other
			// command in cmd/guardrails does. OrgUnitBudgetGroupPreviewRequest
			// declares both parentOrgUnitId and teamId required, and teamId is
			// a body property; the shared client's auto-injection puts teamId
			// in the QUERY STRING, which does not populate a body. Stated
			// honestly: the resulting request therefore carries teamId in both
			// the query (injected by internal/api resolveURL) and the body
			// (explicit here). That duplication is the client's existing
			// behaviour, not something this command introduces (D-31-26).
			body := map[string]interface{}{
				"parentOrgUnitId": parentOrgUnitID,
				"teamId":          cmd.APIClient.TeamID,
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", orgUnitGroupPreviewPath, body, &result); err != nil {
				// 422 is the documented "the two feature flags are not enabled
				// for this team" answer. A bare "Request failed (HTTP 422)"
				// sends an operator to support, so the flags are named here.
				//
				// This is an error-message ENRICHMENT, not a client-side
				// precondition: the CLI cannot observe the flags' state, so it
				// does not pre-refuse a call the server might accept, and the
				// enrichment cannot go stale the moment the flags are turned
				// on. The condition is knowable from the spec in advance,
				// which is why it ships with the command rather than after a
				// support ticket (D-31-27).
				//
				// The original error is wrapped with %w so
				// internal/errors.ExitCodeFor's errors.As still recovers the
				// APIError and resolves the same exit code it would have
				// resolved for the bare relay.
				var apiErr *apierrors.APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == 422 {
					return fmt.Errorf(
						"%w\nThis preview requires both the org-unit-budgets-enabled and "+
							"org-unit-attribution-enabled feature flags to be enabled for this team. "+
							"Contact Revenium support to enable them.", err)
				}
				return err
			}
			return renderOrgUnitGroupPreview(result)
		},
	}

	// The usage string names the value as the RAW numeric org-unit id and
	// contrasts it with hashids on purpose: every other id in this CLI is a
	// Revenium hashid, and an operator who pastes one here gets a 400 or a 404
	// with no hint why (D-31-25). The source is the parentOrgUnitId property
	// description: "as its RAW numeric id (stringified) — see
	// CostControlResource.orgUnitId's doc for why org-unit ids are never
	// hashed, unlike teamId above."
	//
	// The value is deliberately NOT checked as digits-only client-side. The
	// schema types it as a string, so a numeric pattern would be our own
	// invention refusing a value the server would accept. validate.ResourceID
	// (invoked from PreRunE above) is the safety check, and it is exactly
	// that: it rejects empty, control characters, ?, &, #, % and ../, and
	// accepts both "40" and a hashid-shaped value.
	c.Flags().StringVar(&parentOrgUnitID, "parent-org-unit-id", "",
		"RAW numeric org unit id, stringified (e.g. 40) — NOT a Revenium hashid, unlike every other id in this CLI")
	_ = c.MarkFlagRequired("parent-org-unit-id")

	return c
}

// orgUnitTargetsTableDef is the ID | Label table for the targets[] section.
// targets[] elements are ResourceMetadata, whose remaining declared properties
// (resourceType, created, updated, _links) are constant, redundant or HAL noise
// for a list of an org unit's own sub-teams. They round-trip losslessly under
// --json. There is no status column.
var orgUnitTargetsTableDef = output.TableDef{
	Headers:      []string{"ID", "Label"},
	StatusColumn: -1,
}

// renderOrgUnitGroupPreview renders OrgUnitBudgetGroupPreviewResult as two
// sections — the count sentence, then the sub-teams table — or, in JSON mode,
// as the server's result untouched.
//
// WHY THIS FUNCTION BRANCHES ON OUTPUT MODE WHEN A SINGLE-TABLE RENDERER MUST
// NOT. The standing rule (cmd/jobs/roi.go, restated in
// cmd/jobs/economics_render.go) is that a render helper must not branch on
// JSON mode, because the shared render entry point dispatches table-vs-JSON
// internally and a second branch would drift. That rule is correct for a
// SINGLE-table render. A multi-section render cannot use that dispatch:
// calling the entry point once per section emits one JSON document per section
// in JSON mode, and passing a nil payload on the extras emits JSON nulls. So
// this function opens with exactly ONE branch, at the top, and everything below
// it is the table path by construction. No section may consult output mode a
// second time.
//
// THE HONEST CONSEQUENCE, RECORDED RATHER THAN CLAIMED AWAY: the section table
// below the branch goes through the direct table renderer, which does not apply
// the --fields filter. So --fields narrows nothing for this command. That is
// the same gap three prior commands carry; its general repair is an already
// open deferred item and is deliberately NOT attempted here. Any README or
// help prose written for org-unit-group-preview must not claim --fields
// narrows these sections.
func renderOrgUnitGroupPreview(result map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(orgUnitTargetsTableDef, nil, result)
	}

	w := cmd.Output.Writer()

	targets, _ := result["targets"].([]interface{})

	// Zero is a real answer about the org structure — an org unit with no
	// direct sub-teams — not an empty result set. It reads as a sentence, in
	// the register of "absent is one fact about the contract, not five blank
	// rows" (cmd/jobs/economics_render.go's Monetization / not declared
	// precedent), and emits no header row.
	//
	// THE GATE IS ONE TERM, AND THAT TERM IS THE DECODED LIST (code review
	// CR-01, GRDR-08). It previously ORed the server's count into the same
	// condition, so a 200 that carried real sub-teams while omitting
	// targetCount printed the exact opposite fact and discarded every row.
	// OrgUnitBudgetGroupPreviewResult declares NEITHER targetCount NOR targets
	// required, so that response is a shape the contract permits, not a
	// malformed one. The list is the evidence this CLI can enumerate — it
	// renders below, row by row, and an operator can read it. A count is a
	// figure the CLI cannot corroborate against anything. When the two
	// disagree the enumerable one decides, because the answer this command
	// exists to give is the one an operator acts on before writing a
	// groupBy: ORG_UNIT rule that caps spend.
	if len(targets) == 0 {
		fmt.Fprintln(w, "This org unit has no direct sub-teams, so a department-scoped budget rule would cap nothing.")
		return nil
	}

	// targetCount is an int32 the server computed. It is rendered as text and
	// nothing else: no summing, no percentage, no rounding, no tie-breaking.
	// The natural instinct is to derive something from it; there is nothing to
	// derive, and inventing arithmetic here would create a precision contract
	// the CLI could then get wrong.
	count := str(result, "targetCount")

	// The fallback exists so the printed figure is never larger or smaller
	// than the list the operator can see immediately below it. An absent key
	// (str returns the empty string) or a zero standing over a non-empty list
	// is unusable; the length of what the server actually sent is not.
	if count == "" || count == "0" {
		count = strconv.Itoa(len(targets))
	}

	fmt.Fprintf(w, "A department-scoped budget rule would independently cap %s sub-teams.\n", count)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Sub-teams")
	return cmd.Output.RenderTable(orgUnitTargetsTableDef, orgUnitTargetRows(targets))
}

// orgUnitTargetRows converts targets[] to ID | Label rows IN THE SERVER'S
// ORDER. targets is a slice, so its order is already deterministic — unlike a
// range over a map, which is the pitfall the row-order comments elsewhere in
// this repo warn about — and the spec states nothing that would justify
// imposing a different one. Do not add an ordering pass here.
//
// A malformed element is skipped rather than panicked on, matching
// renderRule's defensive handling of the filters array.
func orgUnitTargetRows(targets []interface{}) [][]string {
	rows := make([][]string, 0, len(targets))
	for _, item := range targets {
		target, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		rows = append(rows, []string{str(target, "id"), str(target, "label")})
	}
	return rows
}
