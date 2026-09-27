package billing

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newSeatsCmd()) }

// seatsTableDef defines the table layout for the daily seat-utilization census.
//
// D-30-03: there is deliberately no seatsUsed column. The spec states that
// field is "the same figure as monthlyActive, under the name the seat card
// reads" — two columns carrying one number invite a reader to treat them as
// independent measurements and to add or compare them. seatsUsed is not
// dropped: it is still present, verbatim, under --json, so nothing scripted
// against the raw document loses a field.
var seatsTableDef = output.TableDef{
	Headers: []string{
		"Date",
		"Seats Paid",
		"Daily Active",
		"Weekly Active",
		"Monthly Active",
		"Pending Invites",
	},
	StatusColumn: -1,
}

// newSeatsCmd returns `revenium billing seats` (GET /v2/api/billing/seats,
// single-object report — the raw-Do-into-map idiom cmd/billing/vcs_prs.go
// establishes for this package).
//
// This command deliberately declares NO cobra pre-run hook of any kind. The
// hook at cmd/jobs/roi_summary.go:149-190 repoints the shared client at the
// analytics host with bearer auth, process-wide and without restore. Billing
// lives on the platform host with x-api-key (billing.go:18-21), so there is
// nothing for a hook to do here — and adding one would also drag in the
// SAFE-02 self-recursion guard that block needs when the command is executed
// unattached, as every unit test in this package does.
func newSeatsCmd() *cobra.Command {
	var (
		from string
		to   string
	)

	c := &cobra.Command{
		Use:   "seats",
		Short: "Get daily Claude Enterprise seat utilization",
		Args:  cobra.NoArgs,
		Example: `  # Get daily seat utilization for a date range
  revenium billing seats --from 2026-08-01 --to 2026-08-22

  # Get the full response as JSON
  revenium billing seats --from 2026-08-01 --to 2026-08-22 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			if err := requireTeam(); err != nil {
				return err
			}

			// CRITICAL: the keys handed to the setter below are the OAS
			// parameter names, NOT the CLI flag names — and this operation is
			// the ONLY one in this repository that spells its range fromDate/
			// toDate. Every other date-ranged billing and jobs endpoint uses
			// the other spelling, and getting it wrong here produces a 400
			// that reads like a date-format problem.
			//
			// Two properties of this shape are load-bearing and neither is
			// cosmetic:
			//
			//  1. Every parameter name is a STRING LITERAL. The drift audit's
			//     field extractor reads these calls by AST
			//     (tools/coverage-audit/fields.go, handleSet) and a name it
			//     cannot resolve to a literal is a FATAL extraction error, not
			//     a warning — a name built from a variable, a constant or a
			//     range over a map takes the whole audit down.
			//  2. Encoding goes through url.Values, never string
			//     concatenation. An operator-supplied date carrying & or =
			//     would otherwise append a second query key to an
			//     authenticated request.
			//
			// Both flags are marked required below, so the changed-bit gate
			// roi_summary.go uses for its optional params does not apply: the
			// sets are unconditional.
			//
			// The team parameter is NOT set here. The shared client already
			// appends it in resolveURL, and a second explicit set would put
			// the same key on the wire twice.
			qs := url.Values{}
			qs.Set("fromDate", from)
			qs.Set("toDate", to)

			// Decoded as a map rather than a typed struct so a server field
			// this code does not know about still round-trips through --json.
			// Do, not DoList: this operation returns a single object, not an
			// embedded-items envelope. SeatUtilizationResponse_Read declares
			// no _embedded and no _links, so the package's HAL unwrap helper
			// does not apply either.
			path := "/v2/api/billing/seats" + "?" + qs.Encode()
			var resource map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &resource); err != nil {
				return err
			}
			return renderSeats(resource)
		},
	}

	// The dates are passed through UNPARSED and UNREFORMATTED, exactly as the
	// cmd/jobs date flags are. The usage strings are the spec's own parameter
	// descriptions, so --help answers the format question rather than leaving
	// the operator to guess it.
	c.Flags().StringVar(&from, "from", "", "First UTC day to return, inclusive (yyyy-MM-dd)")
	c.Flags().StringVar(&to, "to", "", "Last UTC day to return, inclusive (yyyy-MM-dd)")
	// Both parameters are required: true in the spec. The discarded-error form
	// is repo-wide (cmd/tools/create.go:57-59,
	// cmd/guardrails/budget_rules_create.go:165-171).
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")

	return c
}

// renderSeats renders the daily census.
//
// It opens with exactly ONE output-mode branch. Nothing below that line may
// consult output mode a second time — calling Render once per section would
// emit two JSON documents, and a nil payload on one of them would emit a JSON
// null. The resource is never mutated before being handed to Render, so --json
// is the server's document verbatim.
func renderSeats(resource map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(seatsTableDef, nil, resource)
	}

	// The Formatter's writer is io.Discard in quiet mode, so the sentence below
	// needs no extra branch of its own.
	w := cmd.Output.Writer()

	days := objectsAt(resource, "days")
	if len(days) == 0 {
		// A success, not an error: the operation description states that an
		// organization with no Claude Enterprise credential returns an empty
		// list. The JSON arm above has already emitted the server's own
		// document for this case, which carries its own empty array.
		_, err := fmt.Fprintln(w, "No seat utilization data found for this window.")
		return err
	}

	// Deterministic row order. The spec states no ordering for days, so sorting
	// here makes row order a property of OUR output rather than an assumption
	// about a server contract that was never made.
	//
	// This runs entirely BELOW the single output-mode branch at the top, on the
	// local slice objectsAt built — never on the resource handed to Render. A
	// sort above that branch would mean --json stops being the server's
	// document in the server's order, which is that flag's whole contract.
	//
	// str() supplies the key so a missing or non-string date sorts as an empty
	// string rather than panicking; date is the one cell in this table that is
	// genuinely a string, and it is the only non-nullable property the day
	// schema declares.
	sort.SliceStable(days, func(i, j int) bool {
		return str(days[i], "date") < str(days[j], "date")
	})

	rows := make([][]string, 0, len(days))
	for _, day := range days {
		// Every numeric cell goes through num(). None goes through str() (which
		// emits scientific notation at or above 1e6) and none goes through the
		// floatVal/formatCount pair in billing.go, which collapses a withheld
		// figure and a measured zero into the same cell.
		rows = append(rows, []string{
			str(day, "date"),
			num(day, "seatsPaid", "%.0f"),
			num(day, "dailyActive", "%.0f"),
			num(day, "weeklyActive", "%.0f"),
			num(day, "monthlyActive", "%.0f"),
			num(day, "pendingInvites", "%.0f"),
		})
	}
	return cmd.Output.RenderTable(seatsTableDef, rows)
}
