package meteringelements

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newCreateCmd builds `revenium metering-elements create ...`.
//
// Per MeteringElementDefinitionResource_Write, --name and --type are
// required. teamId is writeOnly and auto-injected by cmd.APIClient.DoCreate
// (no --team-id flag needed, same precedent as cmd/organizations).
// --description is optional, gated by c.Flags().Changed.
//
// T-06-13 mitigation: validateType runs immediately after flag parsing,
// BEFORE the dry-run check and BEFORE any HTTP call — an invalid --type
// returns the validation error directly with no dry-run render, no request.
func newCreateCmd() *cobra.Command {
	var name, description, elementType string

	c := &cobra.Command{
		Use:         "create",
		Short:       "Create a new metering element definition",
		Annotations: map[string]string{"mutating": "true"},
		Example: `  # Create a metering element definition
  revenium metering-elements create --name "Tokens" --type NUMBER

  # Create with a description
  revenium metering-elements create --name "Tokens" --type NUMBER --description "Token count"`,
		RunE: func(c *cobra.Command, args []string) error {
			if err := validateType(elementType); err != nil {
				return err
			}

			body := map[string]interface{}{
				"name": name,
				"type": elementType,
			}
			if c.Flags().Changed("description") {
				body["description"] = description
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "metering element", "/v2/api/metering-element-definitions", body)
			}

			var result map[string]interface{}
			// DoCreate auto-injects the writeOnly teamId field.
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/metering-element-definitions", body, &result); err != nil {
				return err
			}
			return renderElement(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Metering element name")
	c.Flags().StringVar(&elementType, "type", "", "Metering element type: STRING or NUMBER")
	c.Flags().StringVar(&description, "description", "", "Metering element description")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("type")

	return c
}
