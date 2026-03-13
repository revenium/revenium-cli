package meter

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newAPIRequestCmd() *cobra.Command {
	var transactionID, sourceID, credential, sourceType, method, resource, contentType, remoteHost, userAgent string
	var requestMessageSize int

	c := &cobra.Command{
		Use:         "api-request",
		Short:       "Meter an API request",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a basic API request
  revenium meter api-request --transaction-id txn-123

  # Meter a request with full details
  revenium meter api-request --transaction-id txn-456 --method POST --resource /api/users --source-id src-789`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"transactionId": transactionID,
			}
			if c.Flags().Changed("source-id") {
				body["sourceId"] = sourceID
			}
			if c.Flags().Changed("credential") {
				body["credential"] = credential
			}
			if c.Flags().Changed("source-type") {
				body["sourceType"] = sourceType
			}
			if c.Flags().Changed("method") {
				body["method"] = method
			}
			if c.Flags().Changed("resource") {
				body["resource"] = resource
			}
			if c.Flags().Changed("content-type") {
				body["contentType"] = contentType
			}
			if c.Flags().Changed("remote-host") {
				body["remoteHost"] = remoteHost
			}
			if c.Flags().Changed("user-agent") {
				body["userAgent"] = userAgent
			}
			if c.Flags().Changed("request-message-size") {
				body["requestMessageSize"] = requestMessageSize
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "api-request", "/v2/apis/requests", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/apis/requests", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique identifier to correlate request and response")
	c.Flags().StringVar(&sourceID, "source-id", "", "Source ID")
	c.Flags().StringVar(&credential, "credential", "", "Subscriber credential")
	c.Flags().StringVar(&sourceType, "source-type", "", "Source type (e.g., AI, SDK_PYTHON, KONG)")
	c.Flags().StringVar(&method, "method", "", "HTTP method (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD)")
	c.Flags().StringVar(&resource, "resource", "", "API resource path")
	c.Flags().StringVar(&contentType, "content-type", "", "Request content type")
	c.Flags().StringVar(&remoteHost, "remote-host", "", "Remote host address")
	c.Flags().StringVar(&userAgent, "user-agent", "", "User agent string")
	c.Flags().IntVar(&requestMessageSize, "request-message-size", 0, "Request message size in bytes")
	_ = c.MarkFlagRequired("transaction-id")

	return c
}
