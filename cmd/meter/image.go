package meter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newImageCmd() *cobra.Command {
	var model, provider, requestTime, responseTime, billingUnit string
	var transactionID, traceId, operationType, operationSubtype string
	var agent, environment, region, organizationName, subscriptionId, productName string
	var modelSource, taskType, resolution, quality, style, format string
	var requestDuration, actualImageCount, requestedImageCount int
	var totalCost float64
	var sourceImageProvided bool
	var agenticJobID, agenticJobName, agenticJobType, agenticJobVersion string
	var squadFlags cmd.SquadFlags

	// New optional fields (METER-03 field parity)
	var costType, parentTransactionId, transactionName, traceType, traceName string
	var errorReason, middlewareSource, sourceTransactionId, skipReason string
	var pricingTier, requestedServiceTier, actualServiceTier, priorityTier string
	var outputResponse, inputMessages string
	var retryNumber, errorCode int
	var revisedPromptProvided, promptsTruncated, billingSkipped bool
	// Nested subscriber (D-08): only --subscriber-id/--subscriber-email are exposed.
	// subscriber.credential (name/value) is deliberately excluded — it could carry a
	// secret-like value into shell history/process listings for a rarely-used
	// metering-test-data field. See 03-03-SUMMARY.md.
	var subscriberID, subscriberEmail string

	c := &cobra.Command{
		Use:         "image",
		Short:       "Meter an AI image operation",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter an image generation
  revenium meter image --model dall-e-3 --provider openai --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:05Z --request-duration 5000 --actual-image-count 1 --billing-unit PER_IMAGE

  # Meter with cost details
  revenium meter image --model dall-e-3 --provider openai --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:05Z --request-duration 5000 --actual-image-count 2 --billing-unit PER_IMAGE --total-cost 0.08 --resolution 1024x1024`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"model":            model,
				"provider":         provider,
				"requestTime":      requestTime,
				"responseTime":     responseTime,
				"requestDuration":  requestDuration,
				"actualImageCount": actualImageCount,
				"billingUnit":      billingUnit,
			}
			if c.Flags().Changed("transaction-id") {
				body["transactionId"] = transactionID
			}
			if c.Flags().Changed("trace-id") {
				body["traceId"] = traceId
			}
			if c.Flags().Changed("operation-type") {
				body["operationType"] = operationType
			}
			if c.Flags().Changed("operation-subtype") {
				body["operationSubtype"] = operationSubtype
			}
			if c.Flags().Changed("total-cost") {
				body["totalCost"] = totalCost
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
			if c.Flags().Changed("model-source") {
				body["modelSource"] = modelSource
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
			if c.Flags().Changed("requested-image-count") {
				body["requestedImageCount"] = requestedImageCount
			}
			if c.Flags().Changed("resolution") {
				body["resolution"] = resolution
			}
			if c.Flags().Changed("quality") {
				body["quality"] = quality
			}
			if c.Flags().Changed("style") {
				body["style"] = style
			}
			if c.Flags().Changed("format") {
				body["format"] = format
			}
			if c.Flags().Changed("source-image-provided") {
				body["sourceImageProvided"] = sourceImageProvided
			}
			if c.Flags().Changed("cost-type") {
				body["costType"] = costType
			}
			if c.Flags().Changed("parent-transaction-id") {
				body["parentTransactionId"] = parentTransactionId
			}
			if c.Flags().Changed("transaction-name") {
				body["transactionName"] = transactionName
			}
			if c.Flags().Changed("trace-type") {
				body["traceType"] = traceType
			}
			if c.Flags().Changed("trace-name") {
				body["traceName"] = traceName
			}
			if c.Flags().Changed("error-reason") {
				body["errorReason"] = errorReason
			}
			if c.Flags().Changed("error-code") {
				body["errorCode"] = errorCode
			}
			if c.Flags().Changed("retry-number") {
				body["retryNumber"] = retryNumber
			}
			if c.Flags().Changed("middleware-source") {
				body["middlewareSource"] = middlewareSource
			}
			if c.Flags().Changed("source-transaction-id") {
				body["sourceTransactionId"] = sourceTransactionId
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
			if c.Flags().Changed("priority-tier") {
				body["priorityTier"] = priorityTier
			}
			if c.Flags().Changed("revised-prompt-provided") {
				body["revisedPromptProvided"] = revisedPromptProvided
			}
			if c.Flags().Changed("prompts-truncated") {
				body["promptsTruncated"] = promptsTruncated
			}
			if c.Flags().Changed("billing-skipped") {
				body["billingSkipped"] = billingSkipped
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
			cmd.ApplySquadFlags(c, body, squadFlags)

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "image", "/v2/ai/images", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/ai/images", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	// Required flags
	c.Flags().StringVar(&model, "model", "", "AI model identifier (e.g., dall-e-3)")
	c.Flags().StringVar(&provider, "provider", "", "AI provider (e.g., openai)")
	c.Flags().StringVar(&requestTime, "request-time", "", "Request timestamp (ISO 8601)")
	c.Flags().StringVar(&responseTime, "response-time", "", "Response timestamp (ISO 8601)")
	c.Flags().IntVar(&requestDuration, "request-duration", 0, "Request duration in milliseconds")
	c.Flags().IntVar(&actualImageCount, "actual-image-count", 0, "Number of images generated")
	c.Flags().StringVar(&billingUnit, "billing-unit", "", "Billing unit (PER_IMAGE, PER_MINUTE, PER_SECOND, PER_CHARACTER, PER_TOKEN, CREDITS)")
	_ = c.MarkFlagRequired("model")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("request-time")
	_ = c.MarkFlagRequired("response-time")
	_ = c.MarkFlagRequired("request-duration")
	_ = c.MarkFlagRequired("actual-image-count")
	_ = c.MarkFlagRequired("billing-unit")

	// Optional flags
	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique transaction identifier")
	c.Flags().StringVar(&traceId, "trace-id", "", "Trace identifier for distributed tracing")
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (CHAT, GENERATE, EMBED, CLASSIFY, SUMMARIZE, TRANSLATE, OTHER, TOOL_CALL, RERANK, SEARCH, MODERATION, VISION, TRANSFORM, GUARDRAIL, AUDIO, VIDEO, IMAGE)")
	c.Flags().StringVar(&operationSubtype, "operation-subtype", "", "Operation subtype")
	c.Flags().Float64Var(&totalCost, "total-cost", 0, "Total cost in USD")
	c.Flags().StringVar(&agent, "agent", "", "Agent identifier")
	c.Flags().StringVar(&environment, "environment", "", "Environment name")
	c.Flags().StringVar(&region, "region", "", "Region identifier")
	c.Flags().StringVar(&organizationName, "organization-name", "", "Organization name")
	c.Flags().StringVar(&subscriptionId, "subscription-id", "", "Subscription ID")
	c.Flags().StringVar(&productName, "product-name", "", "Product name")
	c.Flags().StringVar(&modelSource, "model-source", "", "Model source or routing info")
	c.Flags().StringVar(&taskType, "task-type", "", "Task type classification")
	c.Flags().StringVar(&agenticJobID, "agentic-job-id", "", "Agentic job instance identifier — correlates all AI operations within one job execution")
	c.Flags().StringVar(&agenticJobName, "agentic-job-name", "", "Human-readable agentic job name (UI display, analytics grouping)")
	c.Flags().StringVar(&agenticJobType, "agentic-job-type", "", "Agentic job category/type (normalized to lowercase on ingest)")
	c.Flags().StringVar(&agenticJobVersion, "agentic-job-version", "", "Agentic job definition version")
	c.Flags().IntVar(&requestedImageCount, "requested-image-count", 0, "Number of images requested")
	c.Flags().StringVar(&resolution, "resolution", "", "Image resolution (e.g., 1024x1024)")
	c.Flags().StringVar(&quality, "quality", "", "Image quality setting")
	c.Flags().StringVar(&style, "style", "", "Image style setting")
	c.Flags().StringVar(&format, "format", "", "Image format")
	c.Flags().BoolVar(&sourceImageProvided, "source-image-provided", false, "Whether a source image was provided")
	c.Flags().StringVar(&costType, "cost-type", "", "Cost type (AI)")
	c.Flags().StringVar(&parentTransactionId, "parent-transaction-id", "", "Parent transaction identifier")
	c.Flags().StringVar(&transactionName, "transaction-name", "", "Transaction name")
	c.Flags().StringVar(&traceType, "trace-type", "", "Trace type classification for distributed tracing")
	c.Flags().StringVar(&traceName, "trace-name", "", "Trace name for distributed tracing")
	c.Flags().StringVar(&errorReason, "error-reason", "", "Error reason description")
	c.Flags().IntVar(&errorCode, "error-code", 0, "Error code")
	c.Flags().IntVar(&retryNumber, "retry-number", 0, "Retry attempt number")
	c.Flags().StringVar(&middlewareSource, "middleware-source", "", "Middleware source identifier")
	c.Flags().StringVar(&sourceTransactionId, "source-transaction-id", "", "Source transaction identifier")
	c.Flags().StringVar(&skipReason, "skip-reason", "", "Skip reason (FREE_TIER, RATE_LIMITED, QUOTA_EXCEEDED, CONTENT_POLICY_VIOLATION, CAPACITY_UNAVAILABLE, SERVICE_UNAVAILABLE)")
	c.Flags().StringVar(&pricingTier, "pricing-tier", "", "Pricing tier (STANDARD, BATCH)")
	c.Flags().StringVar(&requestedServiceTier, "requested-service-tier", "", "Requested service tier (e.g. priority, default, flex)")
	c.Flags().StringVar(&actualServiceTier, "actual-service-tier", "", "Actual service tier used (e.g. priority, default, flex)")
	c.Flags().StringVar(&priorityTier, "priority-tier", "", "Priority tier (e.g. best_effort, on_demand, committed)")
	c.Flags().BoolVar(&revisedPromptProvided, "revised-prompt-provided", false, "Whether a revised prompt was provided")
	c.Flags().BoolVar(&promptsTruncated, "prompts-truncated", false, "Whether prompts were truncated")
	c.Flags().BoolVar(&billingSkipped, "billing-skipped", false, "Whether billing was skipped for this event")
	c.Flags().StringVar(&outputResponse, "output-response", "", "Model output/response content")
	c.Flags().StringVar(&inputMessages, "input-messages", "", "Input messages as JSON array")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber identifier")
	c.Flags().StringVar(&subscriberEmail, "subscriber-email", "", "Subscriber email")
	cmd.AddSquadFlags(c, &squadFlags)

	return c
}
