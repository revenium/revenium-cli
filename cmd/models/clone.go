package models

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newCloneCmd builds `revenium models clone <globalModelId> ...`.
//
// Per RESEARCH RES-01 table: POST /v2/api/sources/ai/models/{globalModelId}/clone
// is a sub-resource POST — this uses Do() directly (NOT DoCreate, since the
// path isn't the create root), mirroring cmd/jobs/outcome.go's pattern
// exactly. The request body is an optional pricing-override map
// (AIModelPatchResource_Read); when no override flags are passed, an empty
// map {} is sent. The required teamId query param is auto-appended by
// Client.Do() whenever the client is configured with a TeamID — no flag is
// added for it here.
func newCloneCmd() *cobra.Command {
	var (
		cacheCreation1hrCostPerInputToken float64
		cacheCreationCostPerInputToken    float64
		cacheReadCostPerInputToken        float64
		inputCostPerToken                 float64
		outputCostPerToken                float64
	)

	c := &cobra.Command{
		Use:         "clone <globalModelId>",
		Short:       "Clone a global AI model, optionally overriding pricing",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Clone a global model with no pricing overrides
  revenium models clone global-mdl-1

  # Clone a global model with an input cost override
  revenium models clone global-mdl-1 --input-cost-per-token 0.00004`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/clone", url.PathEscape(id))

			body := map[string]interface{}{}
			if c.Flags().Changed("cache-creation-1hr-cost-per-input-token") {
				body["cacheCreation1hrCostPerInputToken"] = cacheCreation1hrCostPerInputToken
			}
			if c.Flags().Changed("cache-creation-cost-per-input-token") {
				body["cacheCreationCostPerInputToken"] = cacheCreationCostPerInputToken
			}
			if c.Flags().Changed("cache-read-cost-per-input-token") {
				body["cacheReadCostPerInputToken"] = cacheReadCostPerInputToken
			}
			if c.Flags().Changed("input-cost-per-token") {
				body["inputCostPerToken"] = inputCostPerToken
			}
			if c.Flags().Changed("output-cost-per-token") {
				body["outputCostPerToken"] = outputCostPerToken
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "clone", "model", path, body)
			}

			var resp map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", path, body, &resp); err != nil {
				return err
			}
			return renderModel(resp)
		},
	}

	c.Flags().Float64Var(&cacheCreation1hrCostPerInputToken, "cache-creation-1hr-cost-per-input-token", 0, "1-hour cache creation cost per input token override in USD")
	c.Flags().Float64Var(&cacheCreationCostPerInputToken, "cache-creation-cost-per-input-token", 0, "Cache creation cost per input token override in USD")
	c.Flags().Float64Var(&cacheReadCostPerInputToken, "cache-read-cost-per-input-token", 0, "Cache read cost per input token override in USD")
	c.Flags().Float64Var(&inputCostPerToken, "input-cost-per-token", 0, "Input cost per token override in USD")
	c.Flags().Float64Var(&outputCostPerToken, "output-cost-per-token", 0, "Output cost per token override in USD")

	return c
}
