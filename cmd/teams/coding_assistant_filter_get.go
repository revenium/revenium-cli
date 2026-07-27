package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCodingAssistantFilterGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "View coding-assistant filter settings for a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View coding-assistant filter settings
  revenium teams coding-assistant-filter get team-123

  # View as JSON
  revenium teams coding-assistant-filter get team-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/teams/%s/settings/coding-assistant-filter", url.PathEscape(args[0]))

			var settings map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &settings); err != nil {
				return err
			}
			return renderCodingAssistantFilterSettings(settings)
		},
	}
}
