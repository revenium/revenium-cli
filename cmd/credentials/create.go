package credentials

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var label, provider, apiKey, description string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new provider credential",
		Example: `  # Create a credential with label and API key
  revenium credentials create --label "OpenAI Key" --provider openai --api-key "sk-abc123"

  # Create with description
  revenium credentials create --label "Anthropic Key" --provider anthropic --api-key "sk-ant-abc" --description "Production key"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"credentialName": label,
				"provider":       provider,
				"apiKey":         apiKey,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/provider-credentials", body, &result); err != nil {
				return err
			}
			return renderCredential(result)
		},
	}

	c.Flags().StringVar(&label, "label", "", "Credential name")
	c.Flags().StringVar(&provider, "provider", "", "Provider name (e.g., openai, anthropic)")
	c.Flags().StringVar(&apiKey, "api-key", "", "API key value")
	c.Flags().StringVar(&description, "description", "", "Credential description")
	_ = c.MarkFlagRequired("label")
	_ = c.MarkFlagRequired("provider")
	_ = c.MarkFlagRequired("api-key")

	return c
}
