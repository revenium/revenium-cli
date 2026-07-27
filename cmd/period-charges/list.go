package periodcharges

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newListCmd returns the `revenium period-charges list` command.
//
// LANDMINE (06-RESEARCH Pattern 3 / 05-RESEARCH Pitfall 1, carried forward
// from cmd/users/me_period_charges.go): GET /v2/api/period-charges returns
// `_embedded` + `hasMore` (bool) + `cursor` (string|null) — NOT a `page`
// object with `totalPages`. The generic cmd.AddListFlags/ListOptsFromFlags/
// DoList path reads page.totalPages to decide whether to keep paginating and
// would silently stop after page one for this endpoint. This command is
// therefore deliberately NOT wired through that path — it never calls
// cmd.AddListFlags and does not expose --page/--page-size. Instead it uses
// fetchAllPeriodCharges, a dedicated cursor-following loop.
func newListCmd() *cobra.Command {
	var invoiceID, start, end string

	c := &cobra.Command{
		Use:   "list",
		Short: "List period charges",
		Args:  cobra.NoArgs,
		Example: `  # List all period charges
  revenium period-charges list

  # Filter by invoice and time range
  revenium period-charges list --invoice-id inv-123 --start 2026-01-01T00:00:00Z --end 2026-02-01T00:00:00Z`,
		RunE: func(c *cobra.Command, args []string) error {
			extra := url.Values{}
			if c.Flags().Changed("invoice-id") {
				extra.Set("invoiceId", invoiceID)
			}
			if c.Flags().Changed("start") {
				extra.Set("startDate", start)
			}
			if c.Flags().Changed("end") {
				extra.Set("endDate", end)
			}

			items, err := fetchAllPeriodCharges(c, extra)
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
			return cmd.Output.Render(tableDef, toRows(items), items)
		},
	}

	c.Flags().StringVar(&invoiceID, "invoice-id", "", "Filter by invoice id")
	c.Flags().StringVar(&start, "start", "", "Filter by period start (RFC3339 timestamp)")
	c.Flags().StringVar(&end, "end", "", "Filter by period end (RFC3339 timestamp)")
	// Deliberately NOT calling cmd.AddListFlags here — see the LANDMINE note
	// above. grep -n "AddListFlags" cmd/period-charges/list.go must return
	// nothing (06-02-PLAN.md acceptance criteria).
	return c
}

// fetchAllPeriodCharges follows this endpoint's hasMore/cursor pagination
// model (NOT page/totalPages) via a dedicated loop, per 06-RESEARCH Pattern 3
// / 05-RESEARCH Pitfall 1 design (a). extra carries any flag-gated optional
// query params (invoiceId/startDate/endDate) and is re-sent on every page
// request; cursor is appended/updated once hasMore is true.
func fetchAllPeriodCharges(c *cobra.Command, extra url.Values) ([]map[string]interface{}, error) {
	var all []map[string]interface{}
	cursor := ""
	for {
		path := "/v2/api/period-charges"
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
		// A1 (06-RESEARCH Assumptions Log): periodChargeResourceList matches
		// the sibling invoiceResourceList/refundResourceList naming convention.
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
