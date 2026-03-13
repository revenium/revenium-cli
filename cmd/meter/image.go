package meter

import (
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
	c.Flags().StringVar(&operationType, "operation-type", "", "Operation type (IMAGE, GENERATE, VISION, etc.)")
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
	c.Flags().IntVar(&requestedImageCount, "requested-image-count", 0, "Number of images requested")
	c.Flags().StringVar(&resolution, "resolution", "", "Image resolution (e.g., 1024x1024)")
	c.Flags().StringVar(&quality, "quality", "", "Image quality setting")
	c.Flags().StringVar(&style, "style", "", "Image style setting")
	c.Flags().StringVar(&format, "format", "", "Image format")
	c.Flags().BoolVar(&sourceImageProvided, "source-image-provided", false, "Whether a source image was provided")

	return c
}
