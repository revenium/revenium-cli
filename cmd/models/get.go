package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newGetCmd builds `revenium models get <id>`.
//
// Per RESEARCH D-01 (mirrors Phase 4 D-06's `metrics api` dead-endpoint
// pattern exactly): GET /v2/api/sources/ai/models/{id} does not exist in the
// live Revenium platform spec at all — only PATCH/PUT(deprecated)/DELETE
// exist on this path template. This is confirmed dead, not a judgment call;
// there is nothing to fix. The command stays registered (flag, don't delete)
// but returns a clear error before any HTTP call is made.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an AI model by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # models get is not currently available; use lookup or list instead
  revenium models lookup --name gpt-4
  revenium models list`,
		RunE: func(c *cobra.Command, args []string) error {
			return fmt.Errorf("models get is not currently available: the live Revenium API spec has no GET-by-id operation for AI models (checked platform spec on 2026-07-20; only PATCH/PUT/DELETE exist on this path). Use 'revenium models lookup --name <name>' or 'revenium models list' instead. This is flagged in the drift report; see docs/analytics-coverage.md or the Phase 8 drift report.")
		},
	}
}
