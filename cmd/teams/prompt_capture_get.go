package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newPromptCaptureGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <team-id>",
		Short: "View prompt capture settings for a team",
		Args:  cobra.ExactArgs(1),
		Example: `  # View prompt capture settings
  revenium teams prompt-capture get team-123

  # View as JSON
  revenium teams prompt-capture get team-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/teams/%s/settings/prompts", args[0])

			var settings map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &settings); err != nil {
				return err
			}
			return renderPromptSettings(settings)
		},
	}
}
