package jobs

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	rerrors "github.com/revenium/revenium-cli/internal/errors"
)

// init registers the outcome-update subcommand on the parent jobs.Cmd,
// following the established multi-init-per-package pattern (see outcome.go).
func init() {
	Cmd.AddCommand(newOutcomeUpdateCmd())
}

// expectedEntityVersionNotDeclaredNotice is printed on stderr whenever
// --expected-entity-version is explicitly supplied, and never otherwise.
//
// Five facts a future reader needs, the first two copied in form from the
// shipped precedent at cmd/teams/attribution_identity_policy_set.go:45-73:
//
//  1. It goes to the cobra command's error writer, never through cmd.Output —
//     the formatter writes to stdout (internal/output/output.go), so a notice
//     there would break every --json consumer's parse. --quiet does NOT
//     suppress it, deliberately: quiet swaps the formatter's stdout writer for
//     io.Discard and never touches stderr, and an advisory a scripted caller
//     can silence is not an advisory.
//  2. It is emitted BEFORE the body is built and before the dry-run gate, so it
//     reaches the operator on a dry run and on a failed write alike, not only on
//     the runs that happen to succeed.
//  3. The substantive fact: `expectedEntityVersion` is declared only by the
//     2.20.0-SNAPSHOT development platform document. The published 2.19.0
//     production document declares neither
//     `UpdateOutcomeRequest_Read.expectedEntityVersion` nor
//     `JobResource_Read.entityVersion`, so a production host accepts the value
//     and discards it: the update is applied unconditionally and the caller
//     gets neither a guarantee nor an error. That is the V2-14
//     reverse-direction asymmetry class this repo already tracks.
//  4. HARD CONSTRAINT on this constant's VALUE: it must contain no "409"
//     substring and no case-insensitive "conflict" substring. The downstream
//     Hermes SDK captures this command's combined output and classifies
//     failures with `(^|[^0-9])409($|[^0-9])` OR `[Cc]onflict`. This advisory
//     also prints on FAILING runs, so either token in the text would make an
//     auth, network, validation or 500 failure be reported to the operator as a
//     stale-version rejection — our own output converting an occasional
//     consumer misfire into a systematic one. Naming the banned tokens in this
//     doc comment is safe: the guard is
//     TestOutcomeUpdateAdvisoryAvoidsConsumerMatcherTokens, which targets the
//     constant's runtime VALUE and never the file's bytes. A grep over this
//     source file would be a self-invalidating gate, because the 409 arm in
//     RunE below legitimately discusses the status code.
//  5. Its expiry: when a published production specification declares both
//     properties, this constant, its gated print in RunE and the tests that pin
//     it should all be deleted together. That is the whole delete.
const expectedEntityVersionNotDeclaredNotice = "Note: --expected-entity-version is declared only by the " +
	"2.20.0-SNAPSHOT development platform specification. The published 2.19.0 production specification " +
	"declares neither the request field nor the job's entityVersion property, so a host that does not " +
	"declare it accepts this value and discards it silently: the update is applied unconditionally, " +
	"with no optimistic-concurrency guarantee and no error."

// newOutcomeUpdateCmd builds `revenium jobs outcome-update <agenticJobId> --reason <value> [...]`.
//
// This is a DISTINCT command from `jobs outcome` (RES-07 / 05-PATTERNS.md
// Pitfall 2): `outcome` is `POST /v2/api/jobs/{id}/outcome` (report an
// outcome for the first time); `outcome-update` is `PATCH` to the SAME path
// (revise a previously-reported outcome). The two commands' error semantics
// are intentionally different and must not share copy:
//   - outcome-update 409 = concurrent update conflict (optimistic-concurrency
//     retry), NOT "already reported / immutable" like outcome's 409.
//   - outcome-update 422 = no outcome has been reported yet, so there is
//     nothing to update — direct the operator to `jobs outcome` first.
//
// reason is REQUIRED (audit trail). executionStatus/outcomeType/outcomeValue/
// outcomeCurrency/metadata are optional and flag-gated via c.Flags().Changed
// so omitted fields never appear in the PATCH body.
func newOutcomeUpdateCmd() *cobra.Command {
	// Constructor-scoped, never package-scoped: cmd/jobs already carries more
	// than one command, and a package-scoped binding is shared across every
	// command in the package (the live outcome-metrics / facts-append bug).
	var (
		reason                string
		executionStatus       string
		outcomeType           string
		outcomeValue          float64
		outcomeCurrency       string
		metadata              string
		expectedEntityVersion int64
	)

	c := &cobra.Command{
		Use:         "outcome-update <agenticJobId>",
		Short:       "Update a previously-reported job outcome",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Revise an outcome's monetary value
  revenium jobs outcome-update loan-app-12345 --reason "corrected value" --outcome-value 175

  # Revise the execution status and outcome type
  revenium jobs outcome-update loan-app-12345 --reason "manual review completed" --execution-status SUCCESS --outcome-type CONVERTED`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// The changed bit is computed ONCE and drives both effects below —
			// the advisory and the body key — so the two provably cannot
			// diverge. A caller can never send the field without the advisory,
			// and never get the advisory without sending the field.
			sendExpectedEntityVersion := c.Flags().Changed("expected-entity-version")

			// FIRST statement, before the body is built and before the dry-run
			// gate, so the operator sees it on a dry run and on a failed write
			// alike, not only on runs that happen to succeed (the Phase 31
			// precedent's placement).
			//
			// GATED on the flag having been supplied at all — a deliberate
			// divergence from D-31-07, which made the Phase 31 notice
			// unconditional. D-31-07 rejected branching on WHICH VALUE the
			// operator supplied, because that encodes a client-side belief
			// about which server-side gate is live. Branching on whether the
			// operator asked for optimistic concurrency AT ALL is not that: an
			// operator who never passed the flag has nothing to be advised
			// about, and printing on every outcome-update would put a line on
			// stderr for every existing caller, contradicting the
			// byte-identical-behaviour requirement.
			if sendExpectedEntityVersion {
				fmt.Fprintln(c.ErrOrStderr(), expectedEntityVersionNotDeclaredNotice)
			}

			// reason is required and always sent (audit trail).
			body := map[string]interface{}{
				"reason": reason,
			}
			if c.Flags().Changed("execution-status") {
				body["executionStatus"] = executionStatus
			}
			if c.Flags().Changed("outcome-type") {
				body["outcomeType"] = outcomeType
			}
			if c.Flags().Changed("outcome-value") {
				body["outcomeValue"] = outcomeValue
			}
			if c.Flags().Changed("outcome-currency") {
				body["outcomeCurrency"] = outcomeCurrency
			}
			if c.Flags().Changed("metadata") {
				body["metadata"] = metadata
			}
			// Same gate block as every other optional field; guarded by the
			// single local above rather than a second Changed() call.
			//
			// The int64 is assigned RAW. Do not format it to a string and do
			// not widen it to float64: encoding/json renders an int64 as
			// digits, whereas a float64 at or above 1e6 renders in scientific
			// notation and a string renders in quotes — either would be a
			// different wire type than the specification declares.
			if sendExpectedEntityVersion {
				body["expectedEntityVersion"] = expectedEntityVersion
			}

			path := fmt.Sprintf("/v2/api/jobs/%s/outcome", url.PathEscape(id))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "outcome-update", "job", path, body)
			}

			// Sub-resource PATCH: use Do with literal "PATCH" — NOT DoUpdate
			// (DoUpdate does a GET-merge-PUT cycle against the parent resource;
			// this is a direct PATCH to the outcome sub-resource).
			var resp map[string]interface{}
			err := cmd.APIClient.Do(c.Context(), "PATCH", path, body, &resp)
			if err != nil {
				var apiErr *rerrors.APIError
				if stderrors.As(err, &apiErr) {
					// DISTINCT copy from outcome.go's 409 (Pitfall 2) — do NOT
					// reuse "already reported... immutable" here.
					if apiErr.StatusCode == http.StatusConflict {
						return fmt.Errorf("concurrent outcome update detected for job %s; re-fetch and retry", id)
					}
					if apiErr.StatusCode == http.StatusUnprocessableEntity {
						return fmt.Errorf("no outcome has been reported yet for job %s; use 'revenium jobs outcome %s' first", id, id)
					}
				}
				return err
			}
			return renderJob(resp)
		},
	}

	c.Flags().StringVar(&reason, "reason", "", "Reason for the outcome update (required, audit trail)")
	c.Flags().StringVar(&executionStatus, "execution-status", "", "Execution result: SUCCESS, FAILED, or CANCELLED")
	c.Flags().StringVar(&outcomeType, "outcome-type", "", "Business outcome type: CONVERTED, ESCALATED, DEFLECTED, UNSUCCESSFUL, or CUSTOM")
	c.Flags().Float64Var(&outcomeValue, "outcome-value", 0, "Monetary value of the outcome")
	c.Flags().StringVar(&outcomeCurrency, "outcome-currency", "", "Currency code (ISO 4217), defaults to USD")
	c.Flags().StringVar(&metadata, "metadata", "", "Additional metadata as JSON string")
	// Int64Var, not StringVar: cobra rejects a non-integer during flag parsing,
	// before any request is formed (the org-unit-id precedent at
	// cmd/billing/vcs_prs_by_org_unit.go:189). The flag NAME is a published
	// external contract matched as a literal string by a downstream consumer —
	// see the notice constant above; renaming it is a breaking change.
	//
	// This usage string carries the same two-token ban as the notice constant
	// and is asserted under it. The root command sets SilenceUsage
	// (cmd/root.go:135-136), so a failed run prints the error alone and this
	// text cannot reach the consumer's capture today — the ban is applied at
	// zero cost so that a string which is never allowed to carry the tokens
	// cannot start carrying them by accident.
	c.Flags().Int64Var(&expectedEntityVersion, "expected-entity-version", 0,
		"Expected entityVersion of the job's current outcome; sent only when this flag is passed. A host that does not declare the field accepts the value and discards it, applying the update unconditionally")
	_ = c.MarkFlagRequired("reason")

	return c
}
