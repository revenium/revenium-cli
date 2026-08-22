package meter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newAudioCmd() *cobra.Command {
	var model, provider, requestTime, responseTime, billingUnit string
	var transactionID, traceId, operationType, operationSubtype string
	var agent, environment, region, organizationName, subscriptionId, productName string
	var modelSource, taskType, language, responseFormat, voice, audioFormat, quality string
	var sourceLanguage, targetLanguage string
	var requestDuration, characterCount, sampleRate, inputTokenCount, outputTokenCount int
	var inputAudioTokenCount, outputAudioTokenCount int
	var totalCost, durationSeconds, speed float64
	var isRealtime bool
	var agenticJobID, agenticJobName, agenticJobType, agenticJobVersion string
	var costType, parentTransactionId, transactionName string
	var traceType, traceName, errorReason, middlewareSource string
	var skipReason, pricingTier, requestedServiceTier, actualServiceTier string
	var outputResponse, inputMessages string
	var subscriberID, subscriberEmail string
	var errorCode, retryNumber int
	var promptsTruncated, billingSkipped bool
	var squadFlags cmd.SquadFlags
	var ticketID string

	c := &cobra.Command{
		Use:         "audio",
		Short:       "Meter an AI audio operation",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter an audio transcription
  revenium meter audio --model whisper-1 --provider openai --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:10Z --request-duration 10000 --billing-unit PER_SECOND --duration-seconds 120

  # Meter a text-to-speech operation
  revenium meter audio --model tts-1 --provider openai --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:00:03Z --request-duration 3000 --billing-unit PER_CHARACTER --character-count 500`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"model":           model,
				"provider":        provider,
				"requestTime":     requestTime,
				"responseTime":    responseTime,
				"requestDuration": requestDuration,
				"billingUnit":     billingUnit,
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
			if c.Flags().Changed("duration-seconds") {
				body["durationSeconds"] = durationSeconds
			}
			if c.Flags().Changed("input-audio-tokens") {
				body["inputAudioTokenCount"] = inputAudioTokenCount
			}
			if c.Flags().Changed("output-audio-tokens") {
				body["outputAudioTokenCount"] = outputAudioTokenCount
			}
			if c.Flags().Changed("character-count") {
				body["characterCount"] = characterCount
			}
			if c.Flags().Changed("sample-rate") {
				body["sampleRate"] = sampleRate
			}
			if c.Flags().Changed("language") {
				body["language"] = language
			}
			if c.Flags().Changed("response-format") {
				body["responseFormat"] = responseFormat
			}
			if c.Flags().Changed("voice") {
				body["voice"] = voice
			}
			if c.Flags().Changed("speed") {
				body["speed"] = speed
			}
			if c.Flags().Changed("source-language") {
				body["sourceLanguage"] = sourceLanguage
			}
			if c.Flags().Changed("target-language") {
				body["targetLanguage"] = targetLanguage
			}
			if c.Flags().Changed("audio-format") {
				body["audioFormat"] = audioFormat
			}
			if c.Flags().Changed("quality") {
				body["quality"] = quality
			}
			if c.Flags().Changed("input-tokens") {
				body["inputTokenCount"] = inputTokenCount
			}
			if c.Flags().Changed("output-tokens") {
				body["outputTokenCount"] = outputTokenCount
			}
			if c.Flags().Changed("is-realtime") {
				body["isRealtime"] = isRealtime
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
			if c.Flags().Changed("retry-number") {
				body["retryNumber"] = retryNumber
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
			if c.Flags().Changed("middleware-source") {
				body["middlewareSource"] = middlewareSource
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
			// Nested subscriber object: only id/email are exposed as CLI flags.
			// credential.name/credential.value are deliberately excluded (D-08) —
			// a credential value could be a secret leaking into shell history or
			// process listings for a rarely-used metering-test-data field.
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
			cmd.ApplyTicketFlag(c, body, ticketID)

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "audio", "/v2/ai/audio", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/ai/audio", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	// Required flags
	c.Flags().StringVar(&model, "model", "", "AI model identifier (e.g., whisper-1, tts-1)")
	c.Flags().StringVar(&provider, "provider", "", "AI provider (e.g., openai)")
	c.Flags().StringVar(&requestTime, "request-time", "", "Request timestamp (ISO 8601)")
	c.Flags().StringVar(&responseTime, "response-time", "", "Response timestamp (ISO 8601)")
	c.Flags().IntVar(&requestDuration, "request-duration", 0, "Request duration in milliseconds")
	c.Flags().StringVar(&billingUnit, "billing-unit", "", "Billing unit (PER_SECOND, PER_CHARACTER, PER_TOKEN, CREDITS)")
	_ = c.MarkFlagRequired("model")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("request-time")
	_ = c.MarkFlagRequired("response-time")
	_ = c.MarkFlagRequired("request-duration")
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
	c.Flags().Float64Var(&durationSeconds, "duration-seconds", 0, "Audio duration in seconds")
	c.Flags().IntVar(&inputAudioTokenCount, "input-audio-tokens", 0, "Number of input audio tokens")
	c.Flags().IntVar(&outputAudioTokenCount, "output-audio-tokens", 0, "Number of output audio tokens")
	c.Flags().IntVar(&characterCount, "character-count", 0, "Character count for TTS")
	c.Flags().IntVar(&sampleRate, "sample-rate", 0, "Audio sample rate")
	c.Flags().StringVar(&language, "language", "", "Language code")
	c.Flags().StringVar(&responseFormat, "response-format", "", "Response format")
	c.Flags().StringVar(&voice, "voice", "", "Voice identifier for TTS")
	c.Flags().Float64Var(&speed, "speed", 0, "Playback speed multiplier")
	c.Flags().StringVar(&sourceLanguage, "source-language", "", "Source language for translation")
	c.Flags().StringVar(&targetLanguage, "target-language", "", "Target language for translation")
	c.Flags().StringVar(&audioFormat, "audio-format", "", "Audio format")
	c.Flags().StringVar(&quality, "quality", "", "Audio quality setting")
	c.Flags().IntVar(&inputTokenCount, "input-tokens", 0, "Number of input tokens")
	c.Flags().IntVar(&outputTokenCount, "output-tokens", 0, "Number of output tokens")
	c.Flags().BoolVar(&isRealtime, "is-realtime", false, "Whether this is a realtime audio session")
	c.Flags().StringVar(&costType, "cost-type", "", "Cost type (AI)")
	c.Flags().StringVar(&parentTransactionId, "parent-transaction-id", "", "Parent transaction identifier")
	c.Flags().StringVar(&transactionName, "transaction-name", "", "Transaction name")
	c.Flags().IntVar(&retryNumber, "retry-number", 0, "Retry attempt number")
	c.Flags().StringVar(&traceType, "trace-type", "", "Trace type classification for distributed tracing")
	c.Flags().StringVar(&traceName, "trace-name", "", "Trace name")
	c.Flags().StringVar(&errorReason, "error-reason", "", "Error reason")
	c.Flags().IntVar(&errorCode, "error-code", 0, "Error code")
	c.Flags().StringVar(&middlewareSource, "middleware-source", "", "Middleware source")
	c.Flags().BoolVar(&promptsTruncated, "prompts-truncated", false, "Whether prompts were truncated")
	c.Flags().BoolVar(&billingSkipped, "billing-skipped", false, "Whether billing was skipped")
	c.Flags().StringVar(&skipReason, "skip-reason", "", "Skip reason (FREE_TIER, RATE_LIMITED, QUOTA_EXCEEDED, CONTENT_POLICY_VIOLATION, CAPACITY_UNAVAILABLE, SERVICE_UNAVAILABLE)")
	c.Flags().StringVar(&pricingTier, "pricing-tier", "", "Pricing tier (STANDARD, BATCH)")
	c.Flags().StringVar(&requestedServiceTier, "requested-service-tier", "", "Requested service tier")
	c.Flags().StringVar(&actualServiceTier, "actual-service-tier", "", "Actual service tier")
	c.Flags().StringVar(&outputResponse, "output-response", "", "Assistant output/response content")
	c.Flags().StringVar(&inputMessages, "input-messages", "", "User input messages as JSON array")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber identifier")
	c.Flags().StringVar(&subscriberEmail, "subscriber-email", "", "Subscriber email address")
	cmd.AddSquadFlags(c, &squadFlags)
	cmd.AddTicketFlag(c, &ticketID)

	return c
}
