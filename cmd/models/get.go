package models

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an AI model by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a model by ID
  revenium models get abc-123

  # Get a model as JSON
  revenium models get abc-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var model map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/ai/models/"+args[0], nil, &model); err != nil {
				return err
			}
			return renderModel(model)
		},
	}
}
