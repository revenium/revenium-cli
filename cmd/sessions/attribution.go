package sessions

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// attributionTableDef is the four-column INTERSECTION table (D-31-17).
//
// WHY THESE FOUR AND NOT MORE.
//
// The response's field set depends on the CALLER'S CREDENTIAL PLANE, and the
// GET operation description says so: a management-plane caller (a WRITE-scoped
// key, or a JWT) receives every field, including the apiKeyId, createdBy,
// modifiedBy and subscriberEmail write provenance, while a METERING-scoped key
// receives only ticketId, ticketTitle and effectiveFrom — the rest are OMITTED
// from the response entirely rather than returned null.
//
// So a table sized for the wider caller shows the narrower one four
// permanently blank columns. Blank is indistinguishable from missing, and it
// invites exactly the wrong reading: that the platform LOST the provenance,
// rather than that it declined to send it to this credential.
//
// Two alternatives were considered and rejected:
//
//   - A wide table with absent keys blanked. Rejected above: indistinguishable
//     from data loss.
//   - Sniffing which keys are present and switching table definitions. Rejected
//     because it makes the output SHAPE a function of the credential — two
//     operators comparing terminals would see different columns and neither
//     would have any way to know why. A stable column set is worth more than a
//     denser one.
//
// Nothing is lost: --json emits the server's interval objects exactly as it
// sent them — every field of every interval, including the provenance this
// table omits — with only the HAL collection envelope unwrapped. That
// unwrapping costs this losslessness argument nothing, because every
// provenance field is PER-ITEM and per-item fields all survive into the array.
// The table says so in a stdout line on every non-JSON run.
var attributionTableDef = output.TableDef{
	Headers:      []string{"Effective From", "Ticket", "Title", "Splits"},
	StatusColumn: -1,
}

// noAttributionMessage is the empty-state line (D-31-19). A session with no
// recorded attribution is ordinary, not an error.
const noAttributionMessage = "No attribution intervals recorded for this session."

// splitsEmpty is what the Splits column shows when an interval has no split —
// an absent key, an explicit null, or an empty array, which all mean the same
// thing.
const splitsEmpty = "—"

// fullDetailHint is the standing pointer at --json, reused verbatim from
// cmd/billing/claude_code_contributions.go so the narrowed table is honest
// about being narrowed. It goes to stdout in non-JSON mode only; inside a JSON
// document it would corrupt a machine consumer's parse.
const fullDetailHint = "(use --json for full detail)"

func newAttributionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attribution <session-id>",
		Short: "List a session's ticket attribution intervals",
		Long: `List every ticket-attribution interval recorded for a coding-assistant session.

Intervals are returned current-first, in the order the API states as a contract
(effectiveFrom descending), and are rendered in exactly that order — the CLI
does not re-order them.

The table shows the fields every caller receives. A management-plane credential
additionally receives write provenance (apiKeyId, createdBy, modifiedBy,
subscriberEmail); --json emits the intervals as a JSON array carrying those
fields and the full contents of any split. The top-level type does not change
with the amount of data — it is a JSON array in both states — so one script
parses a session with intervals and a session with none without branching on
shape.`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # List a session's intervals, current interval first
  revenium sessions attribution sess-8f2a1c

  # Emit the intervals as a JSON array, including write provenance and splits
  revenium sessions attribution sess-8f2a1c --json`,
		// No Annotations and no dry-run gate: this is a read. The write side of
		// session attribution is a recorded deliberate skip and does not live
		// here (T-31-14).
		//
		// No --team-id flag and no team guard either (D-31-20). The GET declares
		// teamId as required:false, and Client.Do already injects it from the
		// resolved Client.TeamID for every non-bearer platform request
		// (internal/api/client.go resolveURL). A per-command flag would
		// duplicate the root's persistent --team-id; a client-side guard would
		// refuse a call the server accepts. This is the second recorded instance
		// of that rule (D-29-03 is the first), and it strengthens the standing
		// deferred observation that any shared team guard would need a
		// per-endpoint allowlist rather than being applied blanket.
		RunE: func(c *cobra.Command, args []string) error {
			// T-31-10: the positional id has already passed cmd.ValidResourceID
			// (which rejects empty, control characters, ? & # % and ../) and is
			// escaped here, so a crafted id cannot append a query string, escape
			// the path segment, or reach a sibling session resource.
			path := fmt.Sprintf("/v2/api/sessions/%s/attribution", url.PathEscape(args[0]))

			var resource map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resource); err != nil {
				return err
			}

			// Unwrap _embedded.objectList defensively. The 200 schema is
			// CollectionModel, whose items are untyped at the collection level,
			// so every hop is type-asserted and a failed assertion is treated as
			// an empty list rather than as an error — a response shaped
			// differently than expected is the same operator-visible situation
			// as a session with nothing recorded.
			var items []map[string]interface{}
			if embedded, ok := resource["_embedded"].(map[string]interface{}); ok {
				if list, ok := embedded["objectList"].([]interface{}); ok {
					for _, raw := range list {
						if item, ok := raw.(map[string]interface{}); ok {
							items = append(items, item)
						}
					}
				}
			}

			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					// A TYPED empty array, matching the shipped empty-state
					// convention (cmd/jobs/list.go). A machine consumer that
					// ranges the result still succeeds; null would not — and
					// the populated exit below now emits the SAME top-level
					// type, so that claim is true of the whole command rather
					// than of this branch alone (code review CR-01).
					return cmd.Output.RenderJSON([]interface{}{})
				}
				// Written to the formatter's OWN writer, not to
				// c.OutOrStdout(): the formatter substitutes io.Discard in
				// quiet non-JSON mode (internal/output/output.go), so quiet
				// suppression is a property of the formatter rather than of
				// this call site, and a line written past the formatter
				// escapes it (code review WR-03, IN-06). This is the writer
				// cmd/guardrails/org_unit_group_preview.go already uses.
				fmt.Fprintln(cmd.Output.Writer(), noAttributionMessage)
				return nil
			}

			// The hint sits below the empty-state branch on purpose: there is
			// nothing extra to see under --json when the list is empty, so
			// pointing at the flag there would be noise. The IsJSON guard is
			// what keeps it out of a JSON document entirely.
			//
			// Written to the formatter's OWN writer, not to c.OutOrStdout():
			// the formatter substitutes io.Discard in quiet non-JSON mode
			// (internal/output/output.go), so quiet suppression is a property
			// of the formatter rather than of this call site. The hint points
			// at a flag that reveals MORE of a table that --quiet has already
			// suppressed, so a hint surviving quiet mode is a pointer to
			// nothing (code review WR-03, IN-06).
			if !cmd.Output.IsJSON() {
				fmt.Fprintln(cmd.Output.Writer(), fullDetailHint)
			}

			// ONE render call, unconditionally, with no output-mode branch of
			// its own. This is a single-table render and Render dispatches
			// table-vs-JSON internally, so a second branch here would be
			// redundant and would drift — cmd/jobs/roi.go's rule.
			//
			// The two-branch sectioned shape (cmd/jobs/economics_render.go) is
			// correct ONLY where there genuinely are multiple sections, because
			// calling Render once per section emits several JSON documents. It
			// costs --fields honouring for everything below the branch, which is
			// the T-30-11 defect this phase exists not to repeat. There is one
			// section here, so there is one Render.
			//
			// The third argument is the DECODED INTERVAL SLICE, not the raw
			// CollectionModel map, and that argument is the whole of code
			// review finding CR-01 (SESS-01). Handing the renderer the wrapper
			// made this exit a JSON OBJECT while the empty exit above emits a
			// JSON ARRAY, so no single consumer could parse both states:
			// `jq '.[]'` iterated _embedded and _links here and the intervals
			// there. Both exits are now one top-level type and a script
			// branches on neither.
			//
			// Nothing per-item is lost, because these elements ARE the server's
			// own decoded objects, unmodified — apiKeyId, createdBy,
			// modifiedBy, subscriberEmail, ticketId, ticketTitle,
			// effectiveFrom, splits and any per-item _links all survive. That
			// is what keeps the four-column table lossless.
			//
			// Exactly two things are dropped: the _embedded envelope and the
			// collection-level _links map. Neither carries pagination for this
			// operation — the GET declares only sessionId and an optional
			// teamId, so there is no page for a collection link to drive.
			//
			// This is the convention cmd/jobs/list.go and
			// cmd/teams/verified_domains_list.go already follow: a list-shaped
			// read in this CLI emits an array under --json in every response
			// state.
			return cmd.Output.Render(attributionTableDef, toAttributionRows(items), items)
		},
	}
}

// toAttributionRows converts the unwrapped interval slice to table rows,
// IN ORDER.
//
// ROW ORDER IS THE SERVER'S, DELIBERATELY (D-31-16). This is where SC4's
// "current interval first" is delivered, and it is delivered by not
// interfering.
//
// The standing repo rule (D-27-08, D-29-10) is to order rows client-side. That
// rule exists for responses whose order the spec leaves UNSTATED, so that row
// order becomes a property of our output rather than an assumption about a
// contract nobody wrote down. Here the contract exists and is explicit: the
// operation description says the endpoint "returns every recorded attribution
// interval for the session, current interval first (effectiveFrom descending),
// so element 0 is what the session is attributed to now." The departure is
// scoped to this one endpoint for exactly that reason.
//
// Re-ordering would also be actively harmful, not merely redundant.
// effectiveFrom is typed as a bare string on SessionAttributionResource — the
// date-time format appears only on the REQUEST schema — so any client-side
// ordering would be lexicographic over a format the schema does not pin, and
// an interval whose timestamp format differed would be moved AWAY from the
// position the server guarantees.
//
// And rendering server order verbatim is still deterministic:
// _embedded.objectList is a SLICE, whose iteration order is stable. That is the
// distinction from cmd/jobs/economics_render.go's row-order comment, which is
// about ranging a MAP — Go map iteration order is randomized, which is why that
// file builds an explicit ordered slice literal. Nothing of the sort applies
// here.
//
// Do not add an ordering step. TestSessionsAttributionPreservesServerOrder
// asserts the rendered first row IS element 0, against a fixture ordered so
// that a lexicographic ascending pass would reverse it.
func toAttributionRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = []string{
			str(it, "effectiveFrom"),
			str(it, "ticketId"),
			str(it, "ticketTitle"),
			splitsCell(it["splits"]),
		}
	}
	return rows
}

// splitsCell renders the Splits column: the decimal element count for a
// non-empty array, and an em dash for an absent key, an explicit null, or an
// empty array (D-31-18).
//
// splits is a nullable array of weighted members, omitted when the interval has
// no split because the scalar ticketId above is then the whole story. A nested
// array does not go in a table cell. The count tells an operator THAT an
// interval is split, which is what tells them to reach for --json.
//
// Two alternatives were considered and rejected:
//
//   - Flattening the members into the cell (TICKET-1 0.6 / TICKET-2 0.4). One
//     to ten entries of unbounded width; it wraps at any terminal size.
//   - A second section per row. The sectioned shape is for one summary plus one
//     table; one section per row is not a rendering.
//
// The CLI performs no arithmetic on the weights — no summing, no rounding, no
// percentage conversion — so there is no precision contract here to get wrong.
func splitsCell(v interface{}) string {
	if members, ok := v.([]interface{}); ok && len(members) > 0 {
		return strconv.Itoa(len(members))
	}
	return splitsEmpty
}
