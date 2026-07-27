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
	var (
		reason          string
		executionStatus string
		outcomeType     string
		outcomeValue    float64
		outcomeCurrency string
		metadata        string
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
	_ = c.MarkFlagRequired("reason")

	return c
}
