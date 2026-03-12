package credentials

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a credential by ID",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get a credential by ID
  revenium credentials get cred-123

  # Get a credential as JSON
  revenium credentials get cred-123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var credential map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/credentials/"+args[0], nil, &credential); err != nil {
				return err
			}
			return renderCredential(credential)
		},
	}
}
