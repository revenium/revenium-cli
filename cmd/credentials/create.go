package credentials

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var label, provider, credentialType, apiKey string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new provider credential",
		Example: `  # Create a credential with label and API key
  revenium credentials create --label "OpenAI Key" --api-key "sk-abc123"

  # Create with all fields
  revenium credentials create --label "OpenAI Key" --provider openai --credential-type API_KEY --api-key "sk-abc123"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"label": label,
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

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/credentials", body, &result); err != nil {
				return err
			}
			return renderCredential(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Credential label")
	c.Flags().StringVar(&provider, "provider", "", "Provider name (e.g., openai)")
	c.Flags().StringVar(&credentialType, "credential-type", "", "Credential type (e.g., API_KEY)")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key value")
	_ = c.MarkFlagRequired("label")

	return c
}
