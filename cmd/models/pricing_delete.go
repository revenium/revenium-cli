package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

func newPricingDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <model-id> <dimension-id>",
		Short:       "Delete a pricing dimension",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(2), cmd.ValidResourceID),
		Example: `  # Delete a pricing dimension (with confirmation)
  revenium models pricing delete model-123 dim-456

  # Delete without confirmation
  revenium models pricing delete model-123 dim-456 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			dimID := args[1]

			if cmd.DryRun() {
				path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions/%s", modelID, dimID)
				return dryrun.Render(cmd.Output, "delete", "pricing dimension", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("pricing dimension", dimID, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions/%s", modelID, dimID)
			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted pricing dimension %s.\n", dimID)
			}
			return nil
		},
	}

	return c
}
