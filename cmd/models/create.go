package models

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newCreateCmd builds `revenium models create ...`.
//
// Per RESEARCH RES-01 table (AIModelResource schema): the writable required
// fields are name/mode/provider/inputCostPerToken/outputCostPerToken, each
// backed by a MarkFlagRequired flag. Every other writable field is optional
// and gated via c.Flags().Changed so it only appears in the request body
// when the user explicitly passed the flag. readOnly fields (id,
// resourceType, label, created, updated, pricingDimensions, _links) are
// never sent. mode/provider are free-text strings in the schema (no enum) —
// the CLI does not client-side-validate them; the server is the source of
// truth (mirrors jobs outcome's D-02 precedent).
//
// Uses DoCreate (not owner-scoped) — models are not owner-scoped, same
// reasoning as organizations create. DoCreate auto-injects teamId/tenantId,
// so no --team/--team-id flag is added here.
func newCreateCmd() *cobra.Command {
	var (
		name                              string
		mode                              string
		provider                          string
		inputCostPerToken                 float64
		outputCostPerToken                float64
		cacheCreationCostPerInputToken    float64
		cacheCreation1hrCostPerInputToken float64
		cacheReadCostPerInputToken        float64
		supportFunctionCalling            bool
		supportsParallelFunctionCalling   bool
		supportsPromptCaching             bool
		supportsResponseSchema            bool
		supportsSystemMessages            bool
		supportsToolChoice                bool
		supportsVision                    bool
		supportsWebSearch                 bool
	)

	c := &cobra.Command{
		Use:         "create",
		Short:       "Create a new AI model",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a model with required fields only
  revenium models create --name "GPT-4" --mode chat --provider OpenAI --input-cost-per-token 0.00003 --output-cost-per-token 0.00006

  # Create a model that supports vision
  revenium models create --name "GPT-4 Vision" --mode chat --provider OpenAI --input-cost-per-token 0.00003 --output-cost-per-token 0.00006 --supports-vision`,
		RunE: func(c *cobra.Command, args []string) error {
			// Required fields — always present (MarkFlagRequired below).
			body := map[string]interface{}{
				"name":               name,
				"mode":               mode,
				"provider":           provider,
				"inputCostPerToken":  inputCostPerToken,
				"outputCostPerToken": outputCostPerToken,
			}
			if c.Flags().Changed("cache-creation-cost-per-input-token") {
				body["cacheCreationCostPerInputToken"] = cacheCreationCostPerInputToken
			}
			if c.Flags().Changed("cache-creation-1hr-cost-per-input-token") {
				body["cacheCreation1hrCostPerInputToken"] = cacheCreation1hrCostPerInputToken
			}
			if c.Flags().Changed("cache-read-cost-per-input-token") {
				body["cacheReadCostPerInputToken"] = cacheReadCostPerInputToken
			}
			if c.Flags().Changed("support-function-calling") {
				body["supportFunctionCalling"] = supportFunctionCalling
			}
			if c.Flags().Changed("supports-parallel-function-calling") {
				body["supportsParallelFunctionCalling"] = supportsParallelFunctionCalling
			}
			if c.Flags().Changed("supports-prompt-caching") {
				body["supportsPromptCaching"] = supportsPromptCaching
			}
			if c.Flags().Changed("supports-response-schema") {
				body["supportsResponseSchema"] = supportsResponseSchema
			}
			if c.Flags().Changed("supports-system-messages") {
				body["supportsSystemMessages"] = supportsSystemMessages
			}
			if c.Flags().Changed("supports-tool-choice") {
				body["supportsToolChoice"] = supportsToolChoice
			}
			if c.Flags().Changed("supports-vision") {
				body["supportsVision"] = supportsVision
			}
			if c.Flags().Changed("supports-web-search") {
				body["supportsWebSearch"] = supportsWebSearch
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "model", "/v2/api/sources/ai/models", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/sources/ai/models", body, &result); err != nil {
				return err
			}
			return renderModel(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Model name")
	c.Flags().StringVar(&mode, "mode", "", "Model mode (e.g. chat, completion, embedding)")
	c.Flags().StringVar(&provider, "provider", "", "Model provider (e.g. OpenAI, Anthropic)")
	c.Flags().Float64Var(&inputCostPerToken, "input-cost-per-token", 0, "Input cost per token in USD")
	c.Flags().Float64Var(&outputCostPerToken, "output-cost-per-token", 0, "Output cost per token in USD")
	c.Flags().Float64Var(&cacheCreationCostPerInputToken, "cache-creation-cost-per-input-token", 0, "Cache creation cost per input token in USD")
	c.Flags().Float64Var(&cacheCreation1hrCostPerInputToken, "cache-creation-1hr-cost-per-input-token", 0, "1-hour cache creation cost per input token in USD")
	c.Flags().Float64Var(&cacheReadCostPerInputToken, "cache-read-cost-per-input-token", 0, "Cache read cost per input token in USD")
	c.Flags().BoolVar(&supportFunctionCalling, "support-function-calling", false, "Model supports function calling")
	c.Flags().BoolVar(&supportsParallelFunctionCalling, "supports-parallel-function-calling", false, "Model supports parallel function calling")
	c.Flags().BoolVar(&supportsPromptCaching, "supports-prompt-caching", false, "Model supports prompt caching")
	c.Flags().BoolVar(&supportsResponseSchema, "supports-response-schema", false, "Model supports response schema")
	c.Flags().BoolVar(&supportsSystemMessages, "supports-system-messages", false, "Model supports system messages")
	c.Flags().BoolVar(&supportsToolChoice, "supports-tool-choice", false, "Model supports tool choice")
	c.Flags().BoolVar(&supportsVision, "supports-vision", false, "Model supports vision input")
	c.Flags().BoolVar(&supportsWebSearch, "supports-web-search", false, "Model supports web search")

	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("mode")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("input-cost-per-token")
	_ = c.MarkFlagRequired("output-cost-per-token")

	return c
}
