package meter

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newEventCmd() *cobra.Command {
	var transactionID, payloadStr, sourceID, subscriberCredential, sourceType string

	c := &cobra.Command{
		Use:         "event",
		Short:       "Meter a generic event",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Meter a simple event
  revenium meter event --transaction-id txn-123 --payload '{"apiCalls": 100, "storageGB": 15.5}'

  # Meter an event with source and subscriber
  revenium meter event --transaction-id txn-456 --payload '{"computeMinutes": 480}' --source-id src-789 --subscriber-credential cred-abc`,
		RunE: func(c *cobra.Command, args []string) error {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				return fmt.Errorf("--payload must be valid JSON: %w", err)
			}

			body := map[string]interface{}{
				"transactionId": transactionID,
				"payload":       payload,
			}
			if c.Flags().Changed("source-id") {
				body["sourceId"] = sourceID
			}
			if c.Flags().Changed("subscriber-credential") {
				body["subscriberCredential"] = subscriberCredential
			}
			if c.Flags().Changed("source-type") {
				body["sourceType"] = sourceType
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "meter", "event", "/v2/events", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/events", body, &result); err != nil {
				return err
			}
			return renderResponse(result)
		},
	}

	c.Flags().StringVar(&transactionID, "transaction-id", "", "Unique identifier for the metering event")
	c.Flags().StringVar(&payloadStr, "payload", "", "JSON object with key-value pairs representing usage metrics")
	c.Flags().StringVar(&sourceID, "source-id", "", "Source ID for the feature being metered")
	c.Flags().StringVar(&subscriberCredential, "subscriber-credential", "", "Subscriber credential for usage attribution")
	c.Flags().StringVar(&sourceType, "source-type", "", "Source type (e.g., AI, SDK_PYTHON, SDK_JS)")
	_ = c.MarkFlagRequired("transaction-id")
	_ = c.MarkFlagRequired("payload")

	return c
}
