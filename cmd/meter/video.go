package meter

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newVideoCmd() *cobra.Command {
	var model, provider, requestTime, responseTime, billingUnit string
	var transactionID, traceId, operationType, operationSubtype string
	var agent, environment, region, organizationName, subscriptionId, productName string
	var modelSource, taskType, resolution, completionStatus, videoJobId string
	var requestDuration, fps int
	var durationSeconds, totalCost, creditsConsumed, requestedDurationSeconds, creditRate float64
	var asyncOperation bool

	c := &cobra.Command{
		Use:         "video",
		Short:       "Meter an AI video operation",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a video generation
  revenium meter video --model veo --provider google --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:01:00Z --request-duration 60000 --duration-seconds 10 --billing-unit PER_SECOND

  # Meter with cost details
  revenium meter video --model sora --provider openai --request-time 2024-01-15T10:00:00Z --response-time 2024-01-15T10:02:00Z --request-duration 120000 --duration-seconds 30 --billing-unit CREDITS --credits-consumed 50`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"model":           model,
				"provider":        provider,
				"requestTime":     requestTime,
				"responseTime":    responseTime,
				"requestDuration": requestDuration,
				"durationSeconds": durationSeconds,
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
			if c.Flags().Changed("fps") {
				body["fps"] = fps
			}
			if c.Flags().Changed("resolution") {
				body["resolution"] = resolution
			}
			if c.Flags().Changed("credits-consumed") {
				body["creditsConsumed"] = creditsConsumed
			}
			if c.Flags().Changed("video-job-id") {
				body["videoJobId"] = videoJobId
			}
			if c.Flags().Changed("requested-duration-seconds") {
				body["requestedDurationSeconds"] = requestedDurationSeconds
			}
			if c.Flags().Changed("credit-rate") {
				body["creditRate"] = creditRate
			}
			if c.Flags().Changed("async-operation") {
				body["asyncOperation"] = asyncOperation
			}
			if c.Flags().Changed("completion-status") {
				body["completionStatus"] = completionStatus
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "video", "/v2/ai/video", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/ai/video", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	// Required flags
	c.Flags().StringVar(&model, "model", "", "AI model identifier (e.g., veo, sora)")
	c.Flags().StringVar(&provider, "provider", "", "AI provider (e.g., google, openai)")
	c.Flags().StringVar(&requestTime, "request-time", "", "Request timestamp (ISO 8601)")
	c.Flags().StringVar(&responseTime, "response-time", "", "Response timestamp (ISO 8601)")
	c.Flags().IntVar(&requestDuration, "request-duration", 0, "Request duration in milliseconds")
	c.Flags().Float64Var(&durationSeconds, "duration-seconds", 0, "Video duration in seconds")
	c.Flags().StringVar(&billingUnit, "billing-unit", "", "Billing unit (PER_SECOND, CREDITS)")
	_ = c.MarkFlagRequired("model")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("request-time")
	_ = c.MarkFlagRequired("response-time")
	_ = c.MarkFlagRequired("request-duration")
	_ = c.MarkFlagRequired("duration-seconds")
	_ = c.MarkFlagRequired("billing-unit")

	// Optional flags
	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique transaction identifier")
	c.Flags().StringVar(&traceId, "trace-id", "", "Trace identifier for distributed tracing")
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (VIDEO, GENERATE, etc.)")
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
	c.Flags().IntVar(&fps, "fps", 0, "Video frames per second")
	c.Flags().StringVar(&resolution, "resolution", "", "Video resolution")
	c.Flags().Float64Var(&creditsConsumed, "credits-consumed", 0, "Credits consumed")
	c.Flags().StringVar(&videoJobId, "video-job-id", "", "Video job identifier")
	c.Flags().Float64Var(&requestedDurationSeconds, "requested-duration-seconds", 0, "Requested video duration")
	c.Flags().Float64Var(&creditRate, "credit-rate", 0, "Credit rate")
	c.Flags().BoolVar(&asyncOperation, "async-operation", false, "Whether this is an async operation")
	c.Flags().StringVar(&completionStatus, "completion-status", "", "Completion status (SUCCESS, PARTIAL_TIMEOUT, FAILED)")

	return c
}
