package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <agenticJobId>",
		Short: "Get a job by agenticJobId",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a job by its user-supplied agenticJobId
  revenium jobs get loan-app-12345

  # Get a job as JSON
  revenium jobs get loan-app-12345 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/jobs/%s", url.PathEscape(args[0]))
			var job map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &job); err != nil {
				return err
			}
			return renderJob(job)
		},
	}
}
