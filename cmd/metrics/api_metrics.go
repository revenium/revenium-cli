package metrics

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newAPIMetricsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "api",
		Short: "Query API metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query API metrics for last 24 hours
  revenium metrics api

  # Query API metrics with time range
  revenium metrics api --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z`,
		RunE: func(c *cobra.Command, args []string) error {
			// D-06: this endpoint is not present in the live Revenium API spec
			// (checked platform, metering, and analytics specs on 2026-07-20).
			// Kept registered (not deleted) per the flag-don't-delete policy, but
			// returns a clear error before any HTTP call is made.
			return fmt.Errorf("metrics api is not currently available: this endpoint is not present in the live Revenium API spec (checked platform, metering, and analytics specs on 2026-07-20). It is flagged in the drift report; see docs/analytics-coverage.md")
		},
	}

	cmd.AddListFlags(c)
	return c
}
