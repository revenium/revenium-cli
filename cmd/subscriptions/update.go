package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var description, subscriberID, productID string
	var patch bool

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a subscription",
		Args:  cobra.ExactArgs(1),
		Example: `  # Full update (PUT)
  revenium subscriptions update sub-123 --description "Updated" --subscriber-id sub-1

  # Partial update (PATCH)
  revenium subscriptions update sub-123 --patch --description "Only this field"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
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
				return fmt.Errorf("no fields specified to update")
			}

			method := "PUT"
			if patch {
				method = "PATCH"
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), method, "/v2/api/subscriptions/"+id, body, &result); err != nil {
				return err
			}
			return renderSubscription(result)
		},
	}

	c.Flags().BoolVar(&patch, "patch", false, "Use partial update (PATCH) instead of full update (PUT)")
	c.Flags().StringVar(&description, "description", "", "Subscription description")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&productID, "product-id", "", "Product ID")

	return c
}
