package meter

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCompletionCmd() *cobra.Command {
	var model, provider, stopReason, requestTime, completionStartTime, responseTime string
	var transactionID, traceId, modelSource, taskType, operationType, agent, environment, region string
	var organizationName, subscriptionId, productName string
	var inputTokenCount, outputTokenCount, totalTokenCount, reasoningTokenCount int
	var cacheCreationTokenCount, cacheReadTokenCount, requestDuration, timeToFirstToken int
	var totalCost, inputTokenCost, outputTokenCost, temperature float64
	var isStreamed bool

	c := &cobra.Command{
		Use:         "completion",
		Short:       "Meter an AI completion",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a basic completion
  revenium meter completion --model gpt-4 --provider openai --input-tokens 500 --output-tokens 200 --total-tokens 700 --stop-reason END --request-time 2024-01-15T10:00:00Z --completion-start-time 2024-01-15T10:00:01Z --response-time 2024-01-15T10:00:05Z --request-duration 5000 --is-streamed

  # Meter a completion with cost details
  revenium meter completion --model claude-3-opus --provider anthropic --input-tokens 1000 --output-tokens 500 --total-tokens 1500 --stop-reason END --request-time 2024-01-15T10:00:00Z --completion-start-time 2024-01-15T10:00:01Z --response-time 2024-01-15T10:00:10Z --request-duration 10000 --is-streamed --total-cost 0.045`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"model":               model,
				"provider":            provider,
				"inputTokenCount":     inputTokenCount,
				"outputTokenCount":    outputTokenCount,
				"totalTokenCount":     totalTokenCount,
				"stopReason":          stopReason,
				"requestTime":         requestTime,
				"completionStartTime": completionStartTime,
				"responseTime":        responseTime,
				"requestDuration":     requestDuration,
				"isStreamed":          isStreamed,
			}
			if c.Flags().Changed("transaction-id") {
				body["transactionId"] = transactionID
			}
			if c.Flags().Changed("trace-id") {
				body["traceId"] = traceId
			}
			if c.Flags().Changed("model-source") {
				body["modelSource"] = modelSource
			}
			if c.Flags().Changed("reasoning-tokens") {
				body["reasoningTokenCount"] = reasoningTokenCount
			}
			if c.Flags().Changed("cache-creation-tokens") {
				body["cacheCreationTokenCount"] = cacheCreationTokenCount
			}
			if c.Flags().Changed("cache-read-tokens") {
				body["cacheReadTokenCount"] = cacheReadTokenCount
			}
			if c.Flags().Changed("total-cost") {
				body["totalCost"] = totalCost
			}
			if c.Flags().Changed("input-token-cost") {
				body["inputTokenCost"] = inputTokenCost
			}
			if c.Flags().Changed("output-token-cost") {
				body["outputTokenCost"] = outputTokenCost
			}
			if c.Flags().Changed("time-to-first-token") {
				body["timeToFirstToken"] = timeToFirstToken
			}
			if c.Flags().Changed("temperature") {
				body["temperature"] = temperature
			}
			if c.Flags().Changed("task-type") {
				body["taskType"] = taskType
			}
			if c.Flags().Changed("operation-type") {
				body["operationType"] = operationType
			}
			if c.Flags().Changed("agent") {
				body["agent"] = agent
			}
			if c.Flags().Changed("environment") {
				body["environment"] = environment
			}
			if c.Flags().Changed("region") {
				body["region"] = region
			}
			if c.Flags().Changed("organization-name") {
				body["organizationName"] = organizationName
			}
			if c.Flags().Changed("subscription-id") {
				body["subscriptionId"] = subscriptionId
			}
			if c.Flags().Changed("product-name") {
				body["productName"] = productName
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "completion", "/v2/ai/completions", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/ai/completions", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	// Required flags
	c.Flags().StringVar(&model, "model", "", "AI model identifier (e.g., gpt-4, claude-3-opus)")
	c.Flags().StringVar(&provider, "provider", "", "AI provider (e.g., openai, anthropic)")
	c.Flags().IntVar(&inputTokenCount, "input-tokens", 0, "Number of input tokens consumed")
	c.Flags().IntVar(&outputTokenCount, "output-tokens", 0, "Number of output tokens generated")
	c.Flags().IntVar(&totalTokenCount, "total-tokens", 0, "Total number of tokens")
	c.Flags().StringVar(&stopReason, "stop-reason", "", "Stop reason (END, END_SEQUENCE, TIMEOUT, TOKEN_LIMIT, COST_LIMIT, COMPLETION_LIMIT, ERROR, CANCELLED)")
	c.Flags().StringVar(&requestTime, "request-time", "", "Request timestamp (ISO 8601)")
	c.Flags().StringVar(&completionStartTime, "completion-start-time", "", "Completion start timestamp (ISO 8601)")
	c.Flags().StringVar(&responseTime, "response-time", "", "Response timestamp (ISO 8601)")
	c.Flags().IntVar(&requestDuration, "request-duration", 0, "Request duration in milliseconds")
	c.Flags().BoolVar(&isStreamed, "is-streamed", false, "Whether streaming was used")
	_ = c.MarkFlagRequired("model")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("input-tokens")
	_ = c.MarkFlagRequired("output-tokens")
	_ = c.MarkFlagRequired("total-tokens")
	_ = c.MarkFlagRequired("stop-reason")
	_ = c.MarkFlagRequired("request-time")
	_ = c.MarkFlagRequired("completion-start-time")
	_ = c.MarkFlagRequired("response-time")
	_ = c.MarkFlagRequired("request-duration")

	// Optional flags
	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique transaction identifier")
	c.Flags().StringVar(&traceId, "trace-id", "", "Trace identifier for distributed tracing")
	c.Flags().StringVar(&modelSource, "model-source", "", "Model source or routing info")
	c.Flags().IntVar(&reasoningTokenCount, "reasoning-tokens", 0, "Number of reasoning tokens")
	c.Flags().IntVar(&cacheCreationTokenCount, "cache-creation-tokens", 0, "Number of cache creation tokens")
	c.Flags().IntVar(&cacheReadTokenCount, "cache-read-tokens", 0, "Number of cache read tokens")
	c.Flags().Float64Var(&totalCost, "total-cost", 0, "Total cost in USD")
	c.Flags().Float64Var(&inputTokenCost, "input-token-cost", 0, "Input token cost in USD")
	c.Flags().Float64Var(&outputTokenCost, "output-token-cost", 0, "Output token cost in USD")
	c.Flags().IntVar(&timeToFirstToken, "time-to-first-token", 0, "Time to first token in milliseconds")
	c.Flags().Float64Var(&temperature, "temperature", 0, "Model temperature setting")
	c.Flags().StringVar(&taskType, "task-type", "", "Task type classification")
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (CHAT, GENERATE, EMBED, CLASSIFY, SUMMARIZE, TRANSLATE, OTHER)")
	c.Flags().StringVar(&agent, "agent", "", "Agent identifier")
	c.Flags().StringVar(&environment, "environment", "", "Environment name")
	c.Flags().StringVar(&region, "region", "", "Region identifier")
	c.Flags().StringVar(&organizationName, "organization-name", "", "Organization name")
	c.Flags().StringVar(&subscriptionId, "subscription-id", "", "Subscription ID")
	c.Flags().StringVar(&productName, "product-name", "", "Product name")

	return c
}
