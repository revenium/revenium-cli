package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var (
		teamID                       string
		inputCostPerToken            float64
		outputCostPerToken           float64
		cacheCreationCostPerInToken  float64
		cacheReadCostPerInToken      float64
	)

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update AI model pricing",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update input cost per token
  revenium models update mdl-123 --team-id team-1 --input-cost-per-token 0.003

  # Update multiple pricing fields
  revenium models update mdl-123 --team-id team-1 --input-cost-per-token 0.003 --output-cost-per-token 0.006`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("input-cost-per-token") {
				body["inputCostPerToken"] = inputCostPerToken
			}
			if c.Flags().Changed("output-cost-per-token") {
				body["outputCostPerToken"] = outputCostPerToken
			}
			if c.Flags().Changed("cache-creation-cost-per-input-token") {
				body["cacheCreationCostPerInputToken"] = cacheCreationCostPerInToken
			}
			if c.Flags().Changed("cache-read-cost-per-input-token") {
				body["cacheReadCostPerInputToken"] = cacheReadCostPerInToken
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/sources/ai/models/%s?teamId=%s", id, teamID)
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result); err != nil {
				return err
			}
			return renderModel(result)
		},
	}

	c.Flags().StringVar(&teamID, "team-id", "", "Team ID (required)")
	c.Flags().Float64Var(&inputCostPerToken, "input-cost-per-token", 0, "Input cost per token")
	c.Flags().Float64Var(&outputCostPerToken, "output-cost-per-token", 0, "Output cost per token")
	c.Flags().Float64Var(&cacheCreationCostPerInToken, "cache-creation-cost-per-input-token", 0, "Cache creation cost per input token")
	c.Flags().Float64Var(&cacheReadCostPerInToken, "cache-read-cost-per-input-token", 0, "Cache read cost per input token")
	_ = c.MarkFlagRequired("team-id")

	return c
}
