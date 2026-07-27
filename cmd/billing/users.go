package billing

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newUsersCmd()) }

// usersTableDef defines the table layout for per-user attributed cost rows
// (billing users list). StatusColumn: -1 — no natural status field.
var usersTableDef = output.TableDef{
	Headers:      []string{"Email", "Cost"},
	StatusColumn: -1,
}

// toUserRows converts a slice of billing user cost-attribution maps to
// table rows.
func toUserRows(users []map[string]interface{}) [][]string {
	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{str(u, "userEmail"), formatCost(floatVal(u, "totalCost"))}
	}
	return rows
}

// userDetailTableDef defines the table layout for a single user's cost
// detail (billing users get <email>).
var userDetailTableDef = output.TableDef{
	Headers:      []string{"Email", "# Entries"},
	StatusColumn: -1,
}

// renderUserDetail renders a single user's cost detail as a single-row
// table or JSON.
func renderUserDetail(detail map[string]interface{}) error {
	rows := [][]string{{str(detail, "userEmail"), countStr(detail, "entries")}}
	return cmd.Output.Render(userDetailTableDef, rows, detail)
}

// newUsersGetCmd returns `revenium billing users get <email>` (GET
// /v2/api/billing/users/{userEmail}, operationId getUserCostDetail).
//
// T-06-09 mitigation: the positional arg is an email, not a synthetic ID.
// It is still routed through cmd.ValidResourceID/url.PathEscape unmodified
// — "@" is confirmed NOT in validate.ResourceID's blocklist (control
// chars, ?/&/#, ../, %XX), so ordinary emails pass through while malformed
// input is rejected before any HTTP call.
func newUsersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <email>",
		Short: "Get a user's billing cost detail by email",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a user's cost detail
  revenium billing users get jane@example.com

  # Get a user's cost detail as JSON
  revenium billing users get jane@example.com --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/billing/users/%s", url.PathEscape(args[0]))
			var detail map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &detail); err != nil {
				return err
			}
			return renderUserDetail(detail)
		},
	}
}

// newUsersCmd returns `revenium billing users` (GET /v2/api/billing/users,
// operationId listUserCostAttribution). Bare `billing users` lists —
// matching the plan's "billing users (or billing users list)" wording — and
// `billing users get <email>` is a nested child command (squads list/get
// shape).
func newUsersCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "users",
		Short: "List per-user attributed billing costs",
		Args:  cobra.NoArgs,
		Example: `  # List per-user attributed costs
  revenium billing users

  # Get a specific user's cost detail
  revenium billing users get jane@example.com`,
		RunE: func(c *cobra.Command, args []string) error {
			var users []map[string]interface{}
			if err := cmd.APIClient.DoList(c.Context(), "/v2/api/billing/users", cmd.ListOptsFromFlags(c), &users); err != nil {
				return err
			}
			if len(users) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No billing user cost data found.")
				return nil
			}
			return cmd.Output.Render(usersTableDef, toUserRows(users), users)
		},
	}

	cmd.AddListFlags(c)
	c.AddCommand(newUsersGetCmd())
	return c
}
