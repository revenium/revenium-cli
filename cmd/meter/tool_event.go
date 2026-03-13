package meter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newToolEventCmd() *cobra.Command {
	var toolID, operation, timestamp, errorMessage string
	var transactionID, agent, organizationName, productName, subscriberCredential string
	var workflowId, traceId, usageMetadataStr string
	var durationMs int
	var costUsd float64
	var success bool

	c := &cobra.Command{
		Use:         "tool-event",
		Short:       "Meter a tool event",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a successful tool call
  revenium meter tool-event --tool-id search-api --duration-ms 150 --success --timestamp 2024-01-15T10:00:00Z

  # Meter a failed tool call
  revenium meter tool-event --tool-id db-query --duration-ms 5000 --success=false --timestamp 2024-01-15T10:00:00Z --error-message "connection timeout"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"toolId":     toolID,
				"durationMs": durationMs,
				"success":    success,
				"timestamp":  timestamp,
			}
			if c.Flags().Changed("transaction-id") {
				body["transactionId"] = transactionID
			}
			if c.Flags().Changed("operation") {
				body["operation"] = operation
			}
			if c.Flags().Changed("error-message") {
				body["errorMessage"] = errorMessage
			}
			if c.Flags().Changed("cost-usd") {
				body["costUsd"] = costUsd
			}
			if c.Flags().Changed("agent") {
				body["agent"] = agent
			}
			if c.Flags().Changed("organization-name") {
				body["organizationName"] = organizationName
			}
			if c.Flags().Changed("product-name") {
				body["productName"] = productName
			}
			if c.Flags().Changed("subscriber-credential") {
				body["subscriberCredential"] = subscriberCredential
			}
			if c.Flags().Changed("workflow-id") {
				body["workflowId"] = workflowId
			}
			if c.Flags().Changed("trace-id") {
				body["traceId"] = traceId
			}
			if c.Flags().Changed("usage-metadata") {
				var usageMetadata map[string]interface{}
				if err := json.Unmarshal([]byte(usageMetadataStr), &usageMetadata); err != nil {
					return fmt.Errorf("--usage-metadata must be valid JSON: %w", err)
				}
				body["usageMetadata"] = usageMetadata
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "tool-event", "/v2/tool/events", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/tool/events", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	// Required flags
	c.Flags().StringVar(&toolID, "tool-id", "", "Identifier of the tool being called")
	c.Flags().IntVar(&durationMs, "duration-ms", 0, "Duration of the tool call in milliseconds")
	c.Flags().BoolVar(&success, "success", false, "Whether the tool call was successful")
	c.Flags().StringVar(&timestamp, "timestamp", "", "Timestamp of the tool call (ISO 8601)")
	_ = c.MarkFlagRequired("tool-id")
	_ = c.MarkFlagRequired("duration-ms")
	_ = c.MarkFlagRequired("timestamp")

	// Optional flags
	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique transaction identifier")
	c.Flags().StringVar(&operation, "operation", "", "Operation name")
	c.Flags().StringVar(&errorMessage, "error-message", "", "Error message if the tool call failed")
	c.Flags().Float64Var(&costUsd, "cost-usd", 0, "Cost in USD")
	c.Flags().StringVar(&agent, "agent", "", "Agent identifier")
	c.Flags().StringVar(&organizationName, "organization-name", "", "Organization name")
	c.Flags().StringVar(&productName, "product-name", "", "Product name")
	c.Flags().StringVar(&subscriberCredential, "subscriber-credential", "", "Subscriber credential")
	c.Flags().StringVar(&workflowId, "workflow-id", "", "Workflow identifier")
	c.Flags().StringVar(&traceId, "trace-id", "", "Trace identifier for distributed tracing")
	c.Flags().StringVar(&usageMetadataStr, "usage-metadata", "", "Usage metadata as JSON object")

	return c
}
