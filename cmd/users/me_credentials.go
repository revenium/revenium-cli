package users

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() {
	meCmd.AddCommand(newMeCredentialsCmd())
}

// meCredentialsTableDef intentionally omits externalSecret (T-05-11 —
// Information Disclosure). JSON mode still passes the raw API response
// through (items), per the codebase's existing table-mask/JSON-passthrough
// convention.
var meCredentialsTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "External ID", "Identity Provider"},
	StatusColumn: -1,
}

func meCredentialsToRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "id"),
			str(m, "name"),
			str(m, "externalId"),
			str(m, "identityProvider"),
		}
	}
	return rows
}

// newMeCredentialsCmd returns `revenium users me credentials`. Two-call
// pattern (05-RESEARCH RES-05): resolves the current user id via
// currentUserID, then GETs /v2/api/users/{id}/credentials — there is no
// literal /users/me/credentials path in the spec.
func newMeCredentialsCmd() *cobra.Command {
	var productID string

	c := &cobra.Command{
		Use:   "credentials",
		Short: "List credentials for the current user",
		Args:  cobra.NoArgs,
		Example: `  # List credentials for the current user
  revenium users me credentials

  # Filter by product
  revenium users me credentials --product-id prod-123`,
		RunE: func(c *cobra.Command, args []string) error {
			id, err := currentUserID(c)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/v2/api/users/%s/credentials", url.PathEscape(id))
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
				fmt.Fprintln(c.OutOrStdout(), "No credentials found.")
				return nil
			}
			return cmd.Output.Render(meCredentialsTableDef, meCredentialsToRows(items), items)
		},
	}

	c.Flags().StringVar(&productID, "product-id", "", "Filter by product id")
	cmd.AddListFlags(c)
	return c
}
