package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// init registers the update subcommand on the parent jobs.Cmd.
// Multi-init per package is core Go semantics — composes with the init()
// in jobs.go (list+get) and the future init()s in create.go / delete.go (D-21).
func init() {
	Cmd.AddCommand(newUpdateCmd())
}

// newUpdateCmd builds the `revenium jobs update <agenticJobId>` subcommand.
//
// CRITICAL: this is a HYBRID of cmd/anomalies/update.go (shell) and
// cmd/models/update.go:55-58 (HTTP call). The HTTP call MUST be
// cmd.APIClient.Do(c.Context(), "PATCH", ...) — true HTTP PATCH per D-02,
// D-23 and JOBS-04. DO NOT use DoUpdate (GET+merge+PUT). DO NOT use PUT.
func newUpdateCmd() *cobra.Command {
	var name string

	c := &cobra.Command{
		Use:         "update <agenticJobId>",
		Short:       "Update a job",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a job's name (user-supplied agenticJobId per D-24)
  revenium jobs update loan-app-12345 --name "Process Loan v2"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]

			// Body assembly via Flags().Changed (D-01: only --name in Phase 12).
			// Use make() so Phase 13+ flag additions are mechanical (no literal seed).
			body := make(map[string]interface{})
			if c.Flags().Changed("name") {
				body["name"] = name
			}

			// D-03: error string is verbatim, fires BEFORE any HTTP call.
			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			// D-25: defensive url.PathEscape because agenticJobId is user-supplied.
			path := fmt.Sprintf("/v2/api/jobs/%s", url.PathEscape(id))

			// D-17: dry-run gate.
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "job", path, body)
			}

			// D-02 / D-23 / JOBS-04: TRUE HTTP PATCH — literal method string.
			// NOT DoUpdate (which is GET+merge+PUT). NOT PUT.
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PATCH", path, body, &result); err != nil {
				return err
			}
			return renderJob(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Job name")

	return c
}
