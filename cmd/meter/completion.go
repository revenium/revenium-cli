package meter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCompletionCmd() *cobra.Command {
	var model, provider, stopReason, requestTime, completionStartTime, responseTime string
	var transactionID, traceId, traceType, modelSource, taskType, operationType, agent, environment, region string
	var organizationName, subscriptionId, productName, systemPrompt, inputMessages, outputResponse string
	var agenticJobID, agenticJobName, agenticJobType, agenticJobVersion string
	var inputTokenCount, outputTokenCount, totalTokenCount, reasoningTokenCount int
	var cacheCreationTokenCount, cacheReadTokenCount, requestDuration, timeToFirstToken int
	var totalCost, inputTokenCost, outputTokenCost, temperature float64
	var isStreamed bool
	var squadFlags cmd.SquadFlags

	// New optional fields (Phase 3, METER-01) — full write-schema parity
	var costType, systemFingerprint, errorReason, middlewareSource, operationSubtype string
	var parentTransactionId, transactionName, traceName, skipReason, pricingTier string
	var requestedServiceTier, actualServiceTier, subscriptionTier, codingAssistantAccountUuid string
	var subscriberID, subscriberEmail string
	var mediationLatency, errorCode, retryNumber, cacheCreation5mTokenCount, cacheCreation1hTokenCount int
	var responseQualityScore, cacheCreationTokenCost, cacheReadTokenCost, costMultiplier float64
	var promptsTruncated, billingSkipped bool
	var skillName, skillInvocationTrigger, skillSource, skillKind, skillPluginName, skillMarketplaceName string

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
			if c.Flags().Changed("trace-type") {
				body["traceType"] = traceType
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
			if c.Flags().Changed("agentic-job-id") {
				body["agenticJobId"] = agenticJobID
			}
			if c.Flags().Changed("agentic-job-name") {
				body["agenticJobName"] = agenticJobName
			}
			if c.Flags().Changed("agentic-job-type") {
				body["agenticJobType"] = agenticJobType
			}
			if c.Flags().Changed("agentic-job-version") {
				body["agenticJobVersion"] = agenticJobVersion
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
			if c.Flags().Changed("system-prompt") {
				body["systemPrompt"] = systemPrompt
			}
			if c.Flags().Changed("output-response") {
				body["outputResponse"] = outputResponse
			}
			if c.Flags().Changed("input-messages") {
				var msgs []interface{}
				if err := json.Unmarshal([]byte(inputMessages), &msgs); err != nil {
					return fmt.Errorf("--input-messages must be valid JSON array: %w", err)
				}
				body["inputMessages"] = inputMessages
			}

			// New optional fields (Phase 3, METER-01) — full write-schema parity
			if c.Flags().Changed("response-quality-score") {
				body["responseQualityScore"] = responseQualityScore
			}
			if c.Flags().Changed("cache-creation-token-cost") {
				body["cacheCreationTokenCost"] = cacheCreationTokenCost
			}
			if c.Flags().Changed("cache-read-token-cost") {
				body["cacheReadTokenCost"] = cacheReadTokenCost
			}
			if c.Flags().Changed("cost-type") {
				body["costType"] = costType
			}
			if c.Flags().Changed("mediation-latency") {
				body["mediationLatency"] = mediationLatency
			}
			if c.Flags().Changed("system-fingerprint") {
				body["systemFingerprint"] = systemFingerprint
			}
			if c.Flags().Changed("error-reason") {
				body["errorReason"] = errorReason
			}
			if c.Flags().Changed("error-code") {
				body["errorCode"] = errorCode
			}
			// Nested subscriber object (D-08): assembled only when at least one
			// sub-flag is passed. subscriber.credential.{name,value} are
			// DELIBERATELY excluded — a secret-bearing credential value could
			// leak into shell history/process listings for a rarely-used
			// test-data field. This is an intentional, documented scope
			// decision, not a silent omission (see 03-01-SUMMARY.md).
			if c.Flags().Changed("subscriber-id") || c.Flags().Changed("subscriber-email") {
				subscriber := map[string]interface{}{}
				if c.Flags().Changed("subscriber-id") {
					subscriber["id"] = subscriberID
				}
				if c.Flags().Changed("subscriber-email") {
					subscriber["email"] = subscriberEmail
				}
				body["subscriber"] = subscriber
			}
			if c.Flags().Changed("middleware-source") {
				body["middlewareSource"] = middlewareSource
			}
			if c.Flags().Changed("operation-subtype") {
				body["operationSubtype"] = operationSubtype
			}
			if c.Flags().Changed("parent-transaction-id") {
				body["parentTransactionId"] = parentTransactionId
			}
			if c.Flags().Changed("transaction-name") {
				body["transactionName"] = transactionName
			}
			if c.Flags().Changed("retry-number") {
				body["retryNumber"] = retryNumber
			}
			if c.Flags().Changed("trace-name") {
				body["traceName"] = traceName
			}
			if c.Flags().Changed("prompts-truncated") {
				body["promptsTruncated"] = promptsTruncated
			}
			if c.Flags().Changed("billing-skipped") {
				body["billingSkipped"] = billingSkipped
			}
			if c.Flags().Changed("skip-reason") {
				body["skipReason"] = skipReason
			}
			if c.Flags().Changed("pricing-tier") {
				body["pricingTier"] = pricingTier
			}
			if c.Flags().Changed("requested-service-tier") {
				body["requestedServiceTier"] = requestedServiceTier
			}
			if c.Flags().Changed("actual-service-tier") {
				body["actualServiceTier"] = actualServiceTier
			}
			if c.Flags().Changed("subscription-tier") {
				body["subscriptionTier"] = subscriptionTier
			}
			if c.Flags().Changed("cost-multiplier") {
				body["costMultiplier"] = costMultiplier
			}
			if c.Flags().Changed("coding-assistant-account-uuid") {
				body["codingAssistantAccountUuid"] = codingAssistantAccountUuid
			}
			if c.Flags().Changed("cache-creation-5m-tokens") {
				body["cacheCreation5mTokenCount"] = cacheCreation5mTokenCount
			}
			if c.Flags().Changed("cache-creation-1h-tokens") {
				body["cacheCreation1hTokenCount"] = cacheCreation1hTokenCount
			}

			if c.Flags().Changed("skill-name") {
				body["skillName"] = skillName
			}
			if c.Flags().Changed("skill-invocation-trigger") {
				body["skillInvocationTrigger"] = skillInvocationTrigger
			}
			if c.Flags().Changed("skill-source") {
				body["skillSource"] = skillSource
			}
			if c.Flags().Changed("skill-kind") {
				body["skillKind"] = skillKind
			}
			if c.Flags().Changed("skill-plugin-name") {
				body["skillPluginName"] = skillPluginName
			}
			if c.Flags().Changed("skill-marketplace-name") {
				body["skillMarketplaceName"] = skillMarketplaceName
			}

			cmd.ApplySquadFlags(c, body, squadFlags)

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
	c.Flags().StringVar(&stopReason, "stop-reason", "", "Stop reason (END, END_SEQUENCE, TIMEOUT, TOKEN_LIMIT, COST_LIMIT, COMPLETION_LIMIT, ERROR, CANCELLED, CONTENT_FILTER, TOOL_CALL)")
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
	c.Flags().StringVar(&traceType, "trace-type", "", "Trace type classification for distributed tracing")
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
	c.Flags().StringVar(&agenticJobID, "agentic-job-id", "", "Agentic job instance identifier — correlates all AI operations within one job execution")
	c.Flags().StringVar(&agenticJobName, "agentic-job-name", "", "Human-readable agentic job name (UI display, analytics grouping)")
	c.Flags().StringVar(&agenticJobType, "agentic-job-type", "", "Agentic job category/type (normalized to lowercase on ingest)")
	c.Flags().StringVar(&agenticJobVersion, "agentic-job-version", "", "Agentic job definition version")
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (CHAT, GENERATE, EMBED, CLASSIFY, SUMMARIZE, TRANSLATE, OTHER, TOOL_CALL, RERANK, SEARCH, MODERATION, VISION, TRANSFORM, GUARDRAIL, AUDIO, VIDEO, IMAGE)")
	c.Flags().StringVar(&agent, "agent", "", "Agent identifier")
	c.Flags().StringVar(&environment, "environment", "", "Environment name")
	c.Flags().StringVar(&region, "region", "", "Region identifier")
	c.Flags().StringVar(&organizationName, "organization-name", "", "Organization name")
	c.Flags().StringVar(&subscriptionId, "subscription-id", "", "Subscription ID")
	c.Flags().StringVar(&productName, "product-name", "", "Product name")
	c.Flags().StringVar(&systemPrompt, "system-prompt", "", "System prompt text sent with the completion")
	c.Flags().StringVar(&outputResponse, "output-response", "", "Assistant output/response content")
	c.Flags().StringVar(&inputMessages, "input-messages", "", "User input messages as JSON array")

	// New optional flags (Phase 3, METER-01) — full write-schema parity
	c.Flags().Float64Var(&responseQualityScore, "response-quality-score", 0, "Response quality score")
	c.Flags().Float64Var(&cacheCreationTokenCost, "cache-creation-token-cost", 0, "Cache creation token cost in USD")
	c.Flags().Float64Var(&cacheReadTokenCost, "cache-read-token-cost", 0, "Cache read token cost in USD")
	c.Flags().StringVar(&costType, "cost-type", "", "Cost type (AI)")
	c.Flags().IntVar(&mediationLatency, "mediation-latency", 0, "Mediation latency in milliseconds")
	c.Flags().StringVar(&systemFingerprint, "system-fingerprint", "", "System fingerprint")
	c.Flags().StringVar(&errorReason, "error-reason", "", "Error reason")
	c.Flags().IntVar(&errorCode, "error-code", 0, "Error code")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&subscriberEmail, "subscriber-email", "", "Subscriber email")
	c.Flags().StringVar(&middlewareSource, "middleware-source", "", "Middleware source")
	c.Flags().StringVar(&operationSubtype, "operation-subtype", "", "Operation subtype")
	c.Flags().StringVar(&parentTransactionId, "parent-transaction-id", "", "Parent transaction identifier")
	c.Flags().StringVar(&transactionName, "transaction-name", "", "Transaction name")
	c.Flags().IntVar(&retryNumber, "retry-number", 0, "Retry attempt number")
	c.Flags().StringVar(&traceName, "trace-name", "", "Trace name")
	c.Flags().BoolVar(&promptsTruncated, "prompts-truncated", false, "Whether prompts were truncated")
	c.Flags().BoolVar(&billingSkipped, "billing-skipped", false, "Whether billing was skipped")
	c.Flags().StringVar(&skipReason, "skip-reason", "", "Skip reason (FREE_TIER, RATE_LIMITED, QUOTA_EXCEEDED, CONTENT_POLICY_VIOLATION, CAPACITY_UNAVAILABLE, SERVICE_UNAVAILABLE)")
	c.Flags().StringVar(&pricingTier, "pricing-tier", "", "Pricing tier (STANDARD, BATCH)")
	c.Flags().StringVar(&requestedServiceTier, "requested-service-tier", "", "Requested service tier")
	c.Flags().StringVar(&actualServiceTier, "actual-service-tier", "", "Actual service tier")
	c.Flags().StringVar(&subscriptionTier, "subscription-tier", "", "Subscription tier")
	c.Flags().Float64Var(&costMultiplier, "cost-multiplier", 0, "Cost multiplier")
	c.Flags().StringVar(&codingAssistantAccountUuid, "coding-assistant-account-uuid", "", "Coding assistant account UUID")
	c.Flags().IntVar(&cacheCreation5mTokenCount, "cache-creation-5m-tokens", 0, "Cache creation 5-minute window token count")
	c.Flags().IntVar(&cacheCreation1hTokenCount, "cache-creation-1h-tokens", 0, "Cache creation 1-hour window token count")

	// Skill tracking fields
	c.Flags().StringVar(&skillName, "skill-name", "", "Skill name")
	c.Flags().StringVar(&skillInvocationTrigger, "skill-invocation-trigger", "", "How the skill was invoked")
	c.Flags().StringVar(&skillSource, "skill-source", "", "Skill source")
	c.Flags().StringVar(&skillKind, "skill-kind", "", "Skill kind")
	c.Flags().StringVar(&skillPluginName, "skill-plugin-name", "", "Plugin the skill belongs to")
	c.Flags().StringVar(&skillMarketplaceName, "skill-marketplace-name", "", "Marketplace the skill was obtained from")

	cmd.AddSquadFlags(c, &squadFlags)

	return c
}
