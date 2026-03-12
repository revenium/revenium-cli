package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var description, subscriberID, productID string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscription",
		Example: `  # Create a subscription with a description
  revenium subscriptions create --description "Production API access"

  # Create a subscription with subscriber and product IDs
  revenium subscriptions create --subscriber-id sub-1 --product-id prod-1 --description "API access"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := make(map[string]interface{})

			if c.Flags().Changed("description") {
				body["description"] = description
			}
			if c.Flags().Changed("subscriber-id") {
				body["subscriberId"] = subscriberID
			}
			if c.Flags().Changed("product-id") {
				body["productId"] = productID
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/subscriptions", body, &result); err != nil {
				return err
			}
			return renderSubscription(result)
		},
	}

	c.Flags().StringVar(&description, "description", "", "Subscription description")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&productID, "product-id", "", "Product ID")

	return c
}
