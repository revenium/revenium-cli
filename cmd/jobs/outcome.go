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

// init registers the outcome subcommand on the parent jobs.Cmd. Per Phase 12
// D-21 / CF-12, multi-init per package composes — this init() runs alongside
// the init()s in jobs.go (list+get), create.go, update.go, and delete.go.
func init() {
	Cmd.AddCommand(newOutcomeCmd())
}

// newOutcomeCmd builds `revenium jobs outcome <agenticJobId> --result <value> [...]`.
//
// Per CONTEXT D-01, the 409 immutability error is handled resource-locally
// inside this RunE — NOT in internal/api/client.go. The mapHTTPError there
// is intentionally resource-agnostic; the "outcome already reported" phrasing
// is specific to this verb (other future 409s — like "transaction already
// finalized" — would deserve their own override).
//
// Per CONTEXT D-02, --result is required (cobra MarkFlagRequired) and maps
// to body["executionStatus"] (NOT body["result"] — load-bearing per
// RESEARCH §Pitfall 3). Five additional flags are optional and gated by
// c.Flags().Changed so empty strings never end up in the POST body.
func newOutcomeCmd() *cobra.Command {
	var (
		result          string
		outcomeType     string
		outcomeValue    float64
		outcomeCurrency string
		metadata        string
		reportedBy      string
	)

	c := &cobra.Command{
		Use:         "outcome <agenticJobId>",
		Short:       "Report a job outcome (immutable)",
		Annotations: map[string]string{"mutating": "true"}, // CF-15 / Phase 12 D-16
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Report a successful outcome
  revenium jobs outcome loan-app-12345 --result SUCCESS

  # Report a converted outcome with monetary value and metadata
  revenium jobs outcome loan-app-12345 --result SUCCESS --outcome-type CONVERTED --outcome-value 150 --outcome-currency USD --metadata '{"customer":"acme"}' --reported-by ops@example.com`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// LOAD-BEARING per RESEARCH §Pitfall 3: the CLI flag --result maps
			// to OAS body field "executionStatus". DO NOT write body["result"].
			body := map[string]interface{}{
				"executionStatus": result,
			}
			// CF-16 / Phase 12 D-02 gating pattern. OAS field names (camelCase)
			// per RESEARCH §A1 — flag-name kebab-case → OAS camelCase.
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
			if c.Flags().Changed("reported-by") {
				body["reportedBy"] = reportedBy
			}

			// D-25 / CF-13: defensive PathEscape because id is user-supplied
			// (cobra arg validator already rejects control chars; this is
			// belt-and-suspenders against future regressions).
			path := fmt.Sprintf("/v2/api/jobs/%s/outcome", url.PathEscape(id))

			// CF-15 / Phase 12 D-17: dry-run gate.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "outcome", "job", path, body)
			}

			// Sub-resource POST: use Do with literal "POST" — NOT DoCreate
			// (DoCreate auto-injects to /v2/api/jobs root; sub-resources
			// require Do per RESEARCH §A1). teamId/tenantId are still
			// auto-appended by Client.Do via internal/api/client.go:68-80.
			var resp map[string]interface{}
			err := cmd.APIClient.Do(c.Context(), "POST", path, body, &resp)
			if err != nil {
				// D-01: resource-specific 409 override at the call site.
				// The user-facing message names the job id and suggests
				// the recovery command, constructed entirely CLI-side from
				// the already-validated id (T-13-02-01 mitigation —
				// apiErr.Message is intentionally NOT interpolated).
				var apiErr *rerrors.APIError
				if stderrors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
					return fmt.Errorf("outcome already reported for job %s; outcomes are immutable — use 'revenium jobs get %s' to view", id, id)
				}
				return err
			}
			return renderJob(resp)
		},
	}

	// D-05: long-flag-only convention, kebab-case for resource fields.
	c.Flags().StringVar(&result, "result", "", "Execution result: SUCCESS, FAILED, or CANCELLED (required)")
	c.Flags().StringVar(&outcomeType, "outcome-type", "", "Business outcome type")
	c.Flags().Float64Var(&outcomeValue, "outcome-value", 0, "Monetary value of the outcome")
	c.Flags().StringVar(&outcomeCurrency, "outcome-currency", "", "Currency code (ISO 4217), defaults to USD")
	c.Flags().StringVar(&metadata, "metadata", "", "Additional metadata as JSON string")
	c.Flags().StringVar(&reportedBy, "reported-by", "", "Identifier of who reported the outcome")
	// D-02: only --result is required. The CLI does NOT enforce a client-side
	// enum on the value — server-side validation is the source of truth.
	_ = c.MarkFlagRequired("result")

	return c
}
