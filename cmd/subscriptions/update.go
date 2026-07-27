package subscriptions

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var description, subscriberID, productID, clientEmail, subscriberEmail string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a subscription",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a subscription description
  revenium subscriptions update sub-123 --description "Updated"

  # Update subscriber and product
  revenium subscriptions update sub-123 --subscriber-id sub-1 --product-id prod-1

  # Update the subscriber email
  revenium subscriptions update sub-123 --subscriber-email user@example.com`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			// D-04: the deprecation warning is stderr-only so JSON stdout stays parseable.
			if c.Flags().Changed("client-email") {
				fmt.Fprintln(c.ErrOrStderr(), "Warning: --client-email is deprecated; use --subscriber-email instead.")
			}
			// D-03: when both flags are passed, --subscriber-email wins — the
			// deprecated clientEmailAddress is never sent alongside the primary field.
			switch {
			case c.Flags().Changed("subscriber-email"):
				updates["subscriberEmail"] = subscriberEmail
			case c.Flags().Changed("client-email"):
				updates["clientEmailAddress"] = clientEmail
			}

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

			path := "/v2/api/subscriptions/" + id

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "subscription", path, updates)
			}

			// D-05: this package owns its own GET+normalize+merge+PUT cycle
			// (rather than calling the shared client's DoUpdate) so the
			// email-blanking protection can live here instead of in the
			// shared internal/api/client.go.
			var existing map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &existing); err != nil {
				return err
			}

			// Flatten the nested IDs this resource returns so the PUT body
			// carries flat IDs, mirroring DoUpdate's nested-ID flattening.
			nestedToFlat := map[string]string{
				"client":  "clientId",
				"product": "productId",
				"team":    "teamId",
				"owner":   "ownerId",
			}
			for nested, flat := range nestedToFlat {
				if _, ok := existing[flat]; !ok {
					if obj, ok := existing[nested].(map[string]interface{}); ok {
						if nestedID, ok := obj["id"].(string); ok {
							existing[flat] = nestedID
						}
					}
				}
			}

			// D-05 relocated email backfill: if the caller passed NEITHER
			// --subscriber-email NOR --client-email, and the fetched object
			// has no flat subscriberEmail/clientEmailAddress, derive an email
			// from the nested client object's label so the PUT doesn't blank it.
			_, updatesHasSubscriberEmail := updates["subscriberEmail"]
			_, updatesHasClientEmail := updates["clientEmailAddress"]
			if !updatesHasSubscriberEmail && !updatesHasClientEmail {
				_, existingHasSubscriberEmail := existing["subscriberEmail"]
				_, existingHasClientEmail := existing["clientEmailAddress"]
				if !existingHasSubscriberEmail && !existingHasClientEmail {
					var derivedEmail string
					if label, ok := existing["label"].(string); ok && label != "" {
						if _, hasClient := existing["client"]; hasClient {
							derivedEmail = label
						}
					}
					if derivedEmail != "" {
						updates["subscriberEmail"] = derivedEmail
					}
				}
			}

			for k, v := range updates {
				existing[k] = v
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, existing, &result); err != nil {
				return err
			}
			return renderSubscription(result)
		},
	}

	c.Flags().StringVar(&description, "description", "", "Subscription description")
	c.Flags().StringVar(&subscriberID, "subscriber-id", "", "Subscriber ID")
	c.Flags().StringVar(&productID, "product-id", "", "Product ID")
	c.Flags().StringVar(&subscriberEmail, "subscriber-email", "", "Subscriber email address (primary)")
	c.Flags().StringVar(&clientEmail, "client-email", "", "Client email address (deprecated, use --subscriber-email)")

	return c
}
