package users

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() {
	meCmd.AddCommand(newMePeriodChargesCmd())
}

var mePeriodChargesTableDef = output.TableDef{
	Headers:      []string{"ID", "Amount", "Currency"},
	StatusColumn: -1,
}

func mePeriodChargesToRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, m := range items {
		rows[i] = []string{
			str(m, "id"),
			str(m, "amount"),
			str(m, "currency"),
		}
	}
	return rows
}

// newMePeriodChargesCmd returns `revenium users me period-charges`.
//
// LANDMINE (05-RESEARCH RES-05 / Pitfall 1): this endpoint returns
// `_embedded` + `hasMore` (bool) + `cursor` (string|null) — NOT a `page`
// object with `totalPages`. The generic cmd.AddListFlags/ListOptsFromFlags/
// DoList path reads page.totalPages to decide whether to keep paginating and
// would silently stop after page one for this endpoint. This command is
// therefore deliberately NOT wired through that path — it never calls
// cmd.AddListFlags and does not expose --page/--page-size. Instead it uses
// fetchAllPeriodCharges, a dedicated cursor-following loop.
func newMePeriodChargesCmd() *cobra.Command {
	var productID, start, end string

	c := &cobra.Command{
		Use:   "period-charges",
		Short: "List period charges for the current user",
		Args:  cobra.NoArgs,
		Example: `  # List period charges for the current user
  revenium users me period-charges

  # Filter by product and time range
  revenium users me period-charges --product-id prod-123 --start 2026-01-01T00:00:00Z --end 2026-02-01T00:00:00Z`,
		RunE: func(c *cobra.Command, args []string) error {
			id, err := currentUserID(c)
			if err != nil {
				return err
			}

			extra := url.Values{}
			if c.Flags().Changed("product-id") {
				extra.Set("productId", productID)
			}
			if c.Flags().Changed("start") {
				extra.Set("start", start)
			}
			if c.Flags().Changed("end") {
				extra.Set("end", end)
			}

			items, err := fetchAllPeriodCharges(c, id, extra)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No period charges found.")
				return nil
			}
			return cmd.Output.Render(mePeriodChargesTableDef, mePeriodChargesToRows(items), items)
		},
	}

	c.Flags().StringVar(&productID, "product-id", "", "Filter by product id")
	c.Flags().StringVar(&start, "start", "", "Filter by period start (RFC3339 timestamp)")
	c.Flags().StringVar(&end, "end", "", "Filter by period end (RFC3339 timestamp)")
	// Deliberately NOT calling cmd.AddListFlags here — see the LANDMINE note
	// above. grep -n "AddListFlags" cmd/users/me_period_charges.go must return
	// nothing (05-04-PLAN.md acceptance criteria).
	return c
}

// fetchAllPeriodCharges follows this endpoint's hasMore/cursor pagination
// model (NOT page/totalPages) via a dedicated loop, per 05-RESEARCH Pitfall 1
// design (a). extra carries any flag-gated optional query params
// (productId/start/end) and is re-sent on every page request; cursor is
// appended/updated once hasMore is true.
func fetchAllPeriodCharges(c *cobra.Command, userID string, extra url.Values) ([]map[string]interface{}, error) {
	var all []map[string]interface{}
	cursor := ""
	for {
		path := fmt.Sprintf("/v2/api/users/%s/period-charges", url.PathEscape(userID))
		q := url.Values{}
		for k, v := range extra {
			q[k] = v
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		if len(q) > 0 {
			path += "?" + q.Encode()
		}

		var wrapper map[string]interface{}
		if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &wrapper); err != nil {
			return nil, err
		}

		embedded, _ := wrapper["_embedded"].(map[string]interface{})
		items, _ := embedded["periodChargeResourceList"].([]interface{})
		for _, it := range items {
			if m, ok := it.(map[string]interface{}); ok {
				all = append(all, m)
			}
		}

		hasMore, _ := wrapper["hasMore"].(bool)
		if !hasMore {
			break
		}
		cursor, _ = wrapper["cursor"].(string)
		if cursor == "" {
			break
		}
	}
	return all, nil
}
