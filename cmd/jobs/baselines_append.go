package jobs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// baselineProvenanceValues is the closed three-value set declared by the dev
// spec at schema BaselineRequest.properties.provenance.enum.
//
// It is deliberately NOT shared with Phase 28's PeriodFactEntry and
// OutcomeMetricEntry, whose provenance enum carries four values
// (MEASURED, SELF_REPORTED, DERIVED, ATTESTED). MEASURED is the only member
// the two sets share, so a validator shared across them would wrongly accept
// SELF_REPORTED on a baseline — an outcome-fact word that means nothing here.
// The variable is named for its schema so that distinction survives a future
// reader who sees two provenance enums and assumes one.
//
// This check is also what goes stale if the server ever widens the set. That
// is intended: a rejection here should read as our check being out of date,
// not as a mystery, which is why the error names the values it accepts.
var baselineProvenanceValues = []string{"CUSTOMER_DECLARED", "MEASURED", "SIGNED_OFF"}

// newBaselinesAppendCmd builds `revenium jobs types baselines append <type>` —
// JOBS-14.
//
// The verb is `append`, matching the spec's own summary "Append a pre-AI
// baseline version". `create` is the repo's general convention for a POST, but
// it reads as "make a new thing" and says nothing about what happens to what
// was there before — which is exactly the impression the append-only story
// exists to avoid.
//
// --dry-run is honoured, using the same shape as every other mutating command
// in cmd/ (create.go, update.go, delete.go, outcome.go, outcome_update.go,
// economics_set.go). D-27-06 declined to build a RICH preview here and that
// decision stands — there is no preflight GET and no diff, only the shared
// internal/dryrun footer. What D-27-06 did not decide, and could not, is that
// --dry-run should silently EXECUTE the write: the flag is a ROOT persistent
// flag (cmd/root.go), so cobra accepts it on this command whether or not
// anybody reads it, README promises it previews any mutation without executing
// it, and cmd/schema.go publishes this command's `mutating` annotation to
// automation. "A POST cannot overwrite anything" is not "a POST creates
// nothing permanent" — a baseline version is permanent, the API offers no
// amend and no delete, so an unread --dry-run is a guard failing open on the
// one irreversible write in this phase.
//
// No --file flag, by design: a piped or spreadsheet-exported bulk import is a
// recorded Deferred Idea, and two code paths for nine scalars is the option
// D-27-02 rejected.
func newBaselinesAppendCmd() *cobra.Command {
	var (
		costPerUnit    float64
		minutesPerUnit float64
		qualityRate    float64
		hourlyRate     float64
		currency       string
		provenance     string
		declaredBy     string
		evidenceURL    string
		effectiveFrom  string
	)

	c := &cobra.Command{
		Use:   "append <type>",
		Short: "Append a new immutable pre-AI baseline version to a job type",
		Long: "Append a new immutable pre-AI baseline version to a job type.\n\n" +
			"Baseline versions are permanent. Appending adds a version; every\n" +
			"earlier version stays readable and keeps the values it was declared\n" +
			"with. The appended version becomes the one in force.",
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Annotations: map[string]string{"mutating": "true"},                   // D-16
		Example: `  # Append a baseline version captured from a time-and-motion study
  revenium jobs types baselines append acme-review \
    --cost-per-unit 4.25 --minutes-per-unit 18 --quality-rate 0.955 \
    --hourly-rate 85.50 --currency USD --provenance MEASURED \
    --declared-by ops@acme.example --evidence-url https://acme.example/evidence/7 \
    --effective-from 2026-01-07T00:00:00Z

  # Append with only what was actually measured — omitted fields take the
  # server's documented defaults
  revenium jobs types baselines append acme-review --cost-per-unit 4.25 --effective-from 2026-01-07T00:00:00Z`,
		RunE: func(c *cobra.Command, args []string) error {
			// D-27-10 / SC5: refuse before anything reaches the wire. This
			// must stay the first statement — the endpoint requires teamId
			// and internal/api appends it only when non-empty, so without
			// the guard the request is issued team-less and the failure is
			// silent.
			if err := requireTeam(); err != nil {
				return err
			}

			// G-1 / D11: refuse before anything reaches the wire when
			// --effective-from is absent. The field is required server-side —
			// omitting it returns HTTP 400 with the bare body `effectiveFrom`
			// — but BaselineRequest declares no `required` key, so this is a
			// spec-vs-server divergence confirmed by a live call against the
			// dev host on 2026-09-05, not an inference from the schema.
			//
			// This guard exists so the operator gets a message naming the
			// flag instead of a relayed server error, which is the standard
			// requireTeam() sets for --team-id (SC5). It is deliberately NOT
			// gated on Changed(): absence is precisely the condition it
			// refuses, so a Changed() test would make the guard unreachable.
			if err := requireEffectiveFrom(c); err != nil {
				return err
			}

			// Every field is gated on Changed() so the server never sees a
			// key the operator did not supply. This is what leaves the two
			// documented server-side defaults to the server: provenance
			// ("Defaults to CUSTOMER_DECLARED when omitted") and declaredBy
			// ("Defaults to the calling principal when omitted").
			//
			// No range check is applied to any numeric. BaselineRequest
			// declares no 0-1 constraint on qualityRate, and its currency
			// note is prose rather than an enum, so enforcing either here
			// would invent a rule and could refuse input the API accepts
			// (D-27-11). Ranges, required-ness and URL shape go to the API.
			// The only boundary enforced client-side is the closed
			// provenance set below.
			body := map[string]interface{}{}
			if c.Flags().Changed("cost-per-unit") {
				body["costPerUnit"] = costPerUnit
			}
			if c.Flags().Changed("minutes-per-unit") {
				body["minutesPerUnit"] = minutesPerUnit
			}
			if c.Flags().Changed("quality-rate") {
				body["qualityRate"] = qualityRate
			}
			if c.Flags().Changed("hourly-rate") {
				body["hourlyRate"] = hourlyRate
			}
			if c.Flags().Changed("currency") {
				body["currency"] = currency
			}
			// Validated here, immediately after flag parsing and before any
			// request is issued, per the in-repo precedent — and ONLY inside
			// the flag-was-supplied branch. The field is optional with a
			// documented server default, so validating an unset empty string
			// would refuse a perfectly legal request (Pitfall 3).
			if c.Flags().Changed("provenance") {
				if err := validateBaselineProvenance(provenance); err != nil {
					return err
				}
				body["provenance"] = provenance
			}
			if c.Flags().Changed("declared-by") {
				body["declaredBy"] = declaredBy
			}
			if c.Flags().Changed("evidence-url") {
				body["evidenceUrl"] = evidenceURL
			}
			if c.Flags().Changed("effective-from") {
				body["effectiveFrom"] = effectiveFrom
			}

			// D-25 / CF-13: defensive PathEscape because <type> is
			// user-supplied — see the identical construction in
			// baselines_list.go.
			path := fmt.Sprintf("/v2/api/jobs/types/%s/baselines", url.PathEscape(args[0]))

			// --dry-run wins over everything: it is checked immediately
			// after the path is resolved and returns before any request is
			// issued. The assertion that matters is the POST counter being
			// zero, not that the command exited 0.
			if cmd.DryRun() { // D-17
				return dryrun.Render(cmd.Output, "append", "job type baseline", path, body)
			}

			// The endpoint answers 201. internal/api.Client.Do treats any
			// status below 400 as success and decodes the body, so a 201 is
			// an ordinary decode and never an error path.
			var appended map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", path, body, &appended); err != nil {
				return err
			}

			// The untouched decoded resource is Render's third argument, so
			// --json emits the server's BaselineResource verbatim.
			return cmd.Output.Render(
				baselinesTableDef,
				toBaselineRows([]map[string]interface{}{appended}),
				appended,
			)
		},
	}

	// D-05: long-flag-only convention for resource fields.
	//
	// None of the nine is marked required and none is defaulted client-side.
	// effectiveFrom is the only non-nullable field of the nine, yet
	// BaselineRequest declares no `required` key at all — so required-ness
	// goes to the API, not to cobra. Do not "fix" this into a required flag:
	// a client-side requirement the schema does not state would refuse
	// requests the server accepts.
	c.Flags().Float64Var(&costPerUnit, "cost-per-unit", 0, "Pre-AI cost to produce one unit")
	c.Flags().Float64Var(&minutesPerUnit, "minutes-per-unit", 0, "Pre-AI minutes of human effort per unit")
	c.Flags().Float64Var(&qualityRate, "quality-rate", 0, "Pre-AI quality rate, as the platform defines it")
	c.Flags().Float64Var(&hourlyRate, "hourly-rate", 0, "Fully loaded hourly rate of the human performing the work")
	c.Flags().StringVar(&currency, "currency", "", "ISO currency code for the monetary fields (server default applies when omitted)")
	c.Flags().StringVar(&provenance, "provenance", "", "How the baseline was established: "+strings.Join(baselineProvenanceValues, ", "))
	c.Flags().StringVar(&declaredBy, "declared-by", "", "Who declared this baseline (server defaults to the calling principal)")
	c.Flags().StringVar(&evidenceURL, "evidence-url", "", "Link to the evidence supporting this baseline")
	c.Flags().StringVar(&effectiveFrom, "effective-from", "", "RFC 3339 timestamp the baseline takes effect from")

	return c
}

// validateBaselineProvenance rejects a value outside the closed set declared
// above, naming every legal value — an operator who mistypes should not have
// to open the API spec to find out what was expected. It follows the shape of
// cmd.ValidatePeriod, which the repo already uses for closed platform enums.
//
// Callers must invoke this only when the flag was actually supplied.
func validateBaselineProvenance(p string) error {
	for _, v := range baselineProvenanceValues {
		if p == v {
			return nil
		}
	}
	return fmt.Errorf("invalid --provenance %q: must be one of %s",
		p, strings.Join(baselineProvenanceValues, ", "))
}
