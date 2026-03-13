package sources

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newCreateCmd() *cobra.Command {
	var name, typ, description, version string

	c := &cobra.Command{
		Use:   "create",
		Short:       "Create a new source",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a source
  revenium sources create --name "My API" --type API

  # Create a source with a description
  revenium sources create --name "AI Service" --type AI --description "Production AI service"`,
		RunE: func(c *cobra.Command, args []string) error {
			body := map[string]interface{}{
				"name":    name,
				"type":    typ,
				"version": version,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "source", "/v2/api/sources", body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.DoCreateWithOwner(c.Context(), "/v2/api/sources", body, &result); err != nil {
				return err
			}
			return renderSource(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Source name")
	c.Flags().StringVar(&typ, "type", "", "Source type (e.g., API, AI)")
	c.Flags().StringVar(&description, "description", "", "Source description")
	c.Flags().StringVar(&version, "version", "1.0.0", "Source version")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("type")

	return c
}
