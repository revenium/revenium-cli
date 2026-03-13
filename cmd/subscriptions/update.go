package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var description, subscriberID, productID string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a subscription",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a subscription description
  revenium subscriptions update sub-123 --description "Updated"

  # Update subscriber and product
  revenium subscriptions update sub-123 --subscriber-id sub-1 --product-id prod-1`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("description") {
				updates["description"] = description
			}
			if c.Flags().Changed("subscriber-id") {
				updates["subscriberId"] = subscriberID
			}
			if c.Flags().Changed("product-id") {
				updates["productId"] = productID
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/subscriptions/"+id, updates, &result); err != nil {
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
