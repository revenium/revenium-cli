package users

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() {
	meCmd.AddCommand(newMeSubscriptionsCmd())
}

// meSubscriptionsTableDef reuses cmd/subscriptions' column set (ID / Name /
// Product) but is duplicated locally rather than importing cmd/subscriptions
// (05-PATTERNS RES-05 — planner's call, kept package-local for consistency
// with the other three me-sublist helpers in this file).
var meSubscriptionsTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Product"},
	StatusColumn: -1,
}

func meSubscriptionsToRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "id"),
			str(m, "name"),
			meNestedStr(m, "product", "label"),
		}
	}
	return rows
}

// meNestedStr extracts a string from a nested object, e.g. m["product"]["label"].
func meNestedStr(m map[string]interface{}, outer, inner string) string {
	if obj, ok := m[outer].(map[string]interface{}); ok {
		return str(obj, inner)
	}
	return ""
}

// newMeSubscriptionsCmd returns `revenium users me subscriptions`. Two-call
// pattern (05-RESEARCH RES-05): resolves the current user id via
// currentUserID, then GETs /v2/api/users/{id}/subscriptions.
func newMeSubscriptionsCmd() *cobra.Command {
	var productID string

	c := &cobra.Command{
		Use:   "subscriptions",
		Short: "List subscriptions for the current user",
		Args:  cobra.NoArgs,
		Example: `  # List subscriptions for the current user
  revenium users me subscriptions

  # Filter by product
  revenium users me subscriptions --product-id prod-123`,
		RunE: func(c *cobra.Command, args []string) error {
			id, err := currentUserID(c)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v2/api/users/%s/subscriptions", url.PathEscape(id))
			if c.Flags().Changed("product-id") {
				path += "?productId=" + url.QueryEscape(productID)
			}
			var items []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &items); err != nil {
				return err
			}
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No subscriptions found.")
				return nil
			}
			return cmd.Output.Render(meSubscriptionsTableDef, meSubscriptionsToRows(items), items)
		},
	}

	c.Flags().StringVar(&productID, "product-id", "", "Filter by product id")
	cmd.AddListFlags(c)
	return c
}
