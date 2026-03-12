package credentials

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newUpdateCmd() *cobra.Command {
	var label, provider, credentialType, apiKey string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a provider credential",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update a credential label
  revenium credentials update cred-123 --label "New Label"

  # Update API key
  revenium credentials update cred-123 --api-key "sk-new-key"`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			body := make(map[string]interface{})

			if c.Flags().Changed("label") {
				body["label"] = label
			}
			if c.Flags().Changed("provider") {
				body["provider"] = provider
			}
			if c.Flags().Changed("credential-type") {
				body["credentialType"] = credentialType
			}
			if c.Flags().Changed("api-key") {
				body["apiKey"] = apiKey
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", "/v2/api/credentials/"+id, body, &result); err != nil {
				return err
			}
			return renderCredential(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Credential label")
	c.Flags().StringVar(&provider, "provider", "", "Provider name")
	c.Flags().StringVar(&credentialType, "credential-type", "", "Credential type")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key value")

	return c
}
