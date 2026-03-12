package sources

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

func newCreateCmd() *cobra.Command {
	var name, typ, description string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new source",
		Example: `  # Create a source
  revenium sources create --name "My API" --type API

  # Create a source with a description
  revenium sources create --name "AI Service" --type AI --description "Production AI service"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name": name,
				"type": typ,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", "/v2/api/sources", body, &result); err != nil {
				return err
			}
			return renderSource(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Source name")
	c.Flags().StringVar(&typ, "type", "", "Source type (e.g., API, AI)")
	c.Flags().StringVar(&description, "description", "", "Source description")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("type")

	return c
}
