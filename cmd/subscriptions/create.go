package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var name, description, subscriberID, productID, clientEmail, subscriberEmail string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscription",
		Example: `  # Create a subscription
  revenium subscriptions create --name "API Access" --subscriber-email user@example.com --product-id prod-1

  # Create a subscription with subscriber and product IDs
  revenium subscriptions create --name "API Access" --subscriber-email user@example.com --subscriber-id sub-1 --product-id prod-1`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			// DRIFT-02 D-02/D-03: --subscriber-email is the primary, non-deprecated
			// flag; --client-email is kept for back-compat. One of the two is
			// required client-side (neither is a stricter server-side requirement,
			// but sending neither is guaranteed to 4xx — see 05-RESEARCH.md).
			if !c.Flags().Changed("subscriber-email") && !c.Flags().Changed("client-email") {
				return fmt.Errorf("one of --subscriber-email or --client-email is required")
			}
			// D-04: the deprecation warning is stderr-only so JSON stdout stays parseable.
			if c.Flags().Changed("client-email") {
				fmt.Fprintln(c.ErrOrStderr(), "Warning: --client-email is deprecated; use --subscriber-email instead.")
			}

			body := map[string]interface{}{
				"name":      name,
				"productId": productID,
			}

			// D-03: when both flags are passed, --subscriber-email wins — the
			// deprecated clientEmailAddress is never sent alongside the primary field.
			switch {
			case c.Flags().Changed("subscriber-email"):
				body["subscriberEmail"] = subscriberEmail
			case c.Flags().Changed("client-email"):
				body["clientEmailAddress"] = clientEmail
			}

			if c.Flags().Changed("description") {
				body["description"] = description
			}
			if c.Flags().Changed("subscriber-id") {
				body["subscriberId"] = subscriberID
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "subscription", "/v2/api/subscriptions", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreateWithOwner(c.Context(), "/v2/api/subscriptions", body, &result); err != nil {
				return err
			}
			return renderSubscription(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Subscription name")
	c.Flags().StringVar(&subscriberEmail, "subscriber-email", "", "Subscriber email address (primary)")
	c.Flags().StringVar(&clientEmail, "client-email", "", "Client email address (deprecated, use --subscriber-email)")
	c.Flags().StringVar(&description, "description", "", "Subscription description")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&productID, "product-id", "", "Product ID")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("product-id")

	return c
}
