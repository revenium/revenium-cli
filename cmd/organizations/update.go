package organizations

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newUpdateCmd() *cobra.Command {
	var name, parentID, externalID string

	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an organization (PUT)",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Rename an organization
  revenium organizations update org-123 --name "New Name"

  # Re-parent an organization
  revenium organizations update org-123 --parent-id parent-org-456

  # Set an external identifier
  revenium organizations update org-123 --external-id ext-789`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			updates := make(map[string]interface{})

			if c.Flags().Changed("name") {
				updates["name"] = name
			}
			if c.Flags().Changed("parent-id") {
				updates["parentId"] = parentID
			}
			if c.Flags().Changed("external-id") {
				updates["externalId"] = externalID
			}

			if len(updates) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/organizations/%s", url.PathEscape(id))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "organization", path, updates)
			}

			var result map[string]interface{}
			// GET + merge + PUT semantics (Phase-15-specific deviation from jobs/budget-rules PATCH).
			if err := cmd.APIClient.DoUpdate(c.Context(), path, updates, &result); err != nil {
				return err
			}
			return renderOrg(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Organization name")
	c.Flags().StringVar(&parentID, "parent-id", "", "Parent organization ID")
	c.Flags().StringVar(&externalID, "external-id", "", "External system identifier")

	return c
}
