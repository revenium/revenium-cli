package credentials

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var label, provider, apiKey, description string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a provider credential",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Update a credential name
  revenium credentials update cred-123 --label "New Label"

  # Update API key
  revenium credentials update cred-123 --api-key "sk-new-key"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("label") {
				updates["credentialName"] = label
			}
			if c.Flags().Changed("provider") {
				updates["provider"] = provider
			}
			if c.Flags().Changed("api-key") {
				updates["apiKey"] = apiKey
			}
			if c.Flags().Changed("description") {
				updates["description"] = description
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "credential", "/v2/api/provider-credentials/"+id, updates)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoUpdate(c.Context(), "/v2/api/provider-credentials/"+id, updates, &result); err != nil {
				return err
			}
			return renderCredential(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Credential name")
	c.Flags().StringVar(&provider, "provider", "", "Provider name")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key value")
	c.Flags().StringVar(&description, "description", "", "Credential description")

	return c
}
