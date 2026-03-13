package subscriptions

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name, description, subscriberID, productID, clientEmail string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscription",
		Example: `  # Create a subscription
  revenium subscriptions create --name "API Access" --client-email user@example.com --product-id prod-1

  # Create a subscription with subscriber and product IDs
  revenium subscriptions create --name "API Access" --client-email user@example.com --subscriber-id sub-1 --product-id prod-1`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":               name,
				"clientEmailAddress": clientEmail,
				"productId":          productID,
			}

			if c.Flags().Changed("description") {
				body["description"] = description
			}
			if c.Flags().Changed("subscriber-id") {
				body["subscriberId"] = subscriberID
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreateWithOwner(c.Context(), "/v2/api/subscriptions", body, &result); err != nil {
				return err
			}
			return renderSubscription(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Subscription name")
	c.Flags().StringVar(&clientEmail, "client-email", "", "Client email address")
	c.Flags().StringVar(&description, "description", "", "Subscription description")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&productID, "product-id", "", "Product ID")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("client-email")
	_ = c.MarkFlagRequired("product-id")

	return c
}
