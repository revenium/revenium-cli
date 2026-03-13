package meter

import (
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
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (AUDIO, GENERATE, TRANSLATE, etc.)")
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

	return c
}
