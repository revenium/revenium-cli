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
			// BILL-07 deprecation notice, copied in form from the shipped
			// precedent at cmd/subscriptions/create.go:32-35. It goes to the
			// cobra command's error writer, never through cmd.Output (whose
			// writer is stdout, internal/output/output.go:80-83), so --json
			// stdout stays parseable; and it is emitted BEFORE the request is
			// built so it reaches the operator even when the withdrawn endpoint
			// 404s.
			//
			// The order of the two successors is fixed and load-bearing
			// (D-30-01): this endpoint returned per-user COST detail, so the
			// surviving `revenium billing users` list is the closer redirect for
			// the question the operator was actually asking.
			// `revenium billing vcs-pr-health` answers a different, adjacent
			// question and is named second. Two successors are named because
			// neither one alone replaces the withdrawn endpoint.
			fmt.Fprintln(c.ErrOrStderr(), "Warning: `revenium billing users get` is deprecated; its endpoint is withdrawn upstream. Use `revenium billing users` instead for cost, or `revenium billing vcs-pr-health` for per-engineer activity.")

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
		// D-30-02: this list targets /v2/api/billing/users, which is PRESENT
		// and healthy in the dev platform document at 2.20.0-SNAPSHOT. Only
		// the `get <email>` child's endpoint is withdrawn. Deliberately no
		// deprecation notice here — warning that a working command is going
		// away would misreport the state of the system. Do not "complete"
		// the BILL-07 change by adding one.
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
