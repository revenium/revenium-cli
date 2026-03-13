package teams

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newPromptCaptureSetCmd() *cobra.Command {
	var enabled bool

	c := &cobra.Command{
		Use:   "set <team-id>",
		Short: "Update prompt capture settings for a team",
		Args:  cobra.ExactArgs(1),
		Example: `  # Enable prompt capture
  revenium teams prompt-capture set team-123 --enabled true

  # Set max prompt length
  revenium teams prompt-capture set team-123 --max-prompt-length 4096`,
		RunE: func(c *cobra.Command, args []string) error {
			body := make(map[string]interface{})

			if c.Flags().Changed("enabled") {
				body["promptCaptureEnabled"] = enabled
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/teams/%s/settings/prompts", args[0])

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}
			return renderPromptSettings(result)
		},
	}

	c.Flags().BoolVar(&enabled, "enabled", false, "Enable or disable prompt capture")

	return c
}
