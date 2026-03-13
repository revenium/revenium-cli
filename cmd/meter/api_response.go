package meter

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newAPIResponseCmd() *cobra.Command {
	var transactionID, contentType string
	var responseCode, totalDuration, responseMessageSize int
	var backendLatency, gatewayLatency float64

	c := &cobra.Command{
		Use:         "api-response",
		Short:       "Meter an API response",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a basic API response
  revenium meter api-response --transaction-id txn-123 --response-code 200

  # Meter a response with latency details
  revenium meter api-response --transaction-id txn-456 --response-code 200 --total-duration 150 --backend-latency 120.5`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"transactionId": transactionID,
				"responseCode":  responseCode,
			}
			if c.Flags().Changed("total-duration") {
				body["totalDuration"] = totalDuration
			}
			if c.Flags().Changed("response-message-size") {
				body["responseMessageSize"] = responseMessageSize
			}
			if c.Flags().Changed("content-type") {
				body["contentType"] = contentType
			}
			if c.Flags().Changed("backend-latency") {
				body["backendLatency"] = backendLatency
			}
			if c.Flags().Changed("gateway-latency") {
				body["gatewayLatency"] = gatewayLatency
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "api-response", "/v2/apis/responses", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/apis/responses", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique identifier to correlate request and response")
	c.Flags().IntVar(&responseCode, "response-code", 0, "HTTP response status code")
	c.Flags().IntVar(&totalDuration, "total-duration", 0, "Total duration in milliseconds")
	c.Flags().IntVar(&responseMessageSize, "response-message-size", 0, "Response message size in bytes")
	c.Flags().StringVar(&contentType, "content-type", "", "Response content type")
	c.Flags().Float64Var(&backendLatency, "backend-latency", 0, "Backend latency in milliseconds")
	c.Flags().Float64Var(&gatewayLatency, "gateway-latency", 0, "Gateway latency in milliseconds")
	_ = c.MarkFlagRequired("transaction-id")
	_ = c.MarkFlagRequired("response-code")

	return c
}
