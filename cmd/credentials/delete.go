package credentials

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/resource"
)

func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a provider credential",
		Args:  cobra.ExactArgs(1),
		Example: `  # Delete a credential (with confirmation)
  revenium credentials delete cred-123

  # Delete without confirmation
  revenium credentials delete cred-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("credential", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", "/v2/api/provider-credentials/"+id, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted credential %s.\n", id)
			}
			return nil
		},
	}

	return c
}
