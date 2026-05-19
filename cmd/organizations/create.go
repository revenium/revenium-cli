package organizations

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newCreateCmd builds `revenium organizations create ...`.
//
// Per RESEARCH D-05 LOCKED (verified against OrganizationResource.required),
// only --name is MarkFlagRequired. All other OAS-required fields are either
// readOnly server-set (_links, id, label, resourceType, tenant) or auto-injected
// by cmd.APIClient.DoCreate (tenantId via client config, CF-12-20).
//
// Optional flags --external-id and --parent-id are gated by c.Flags().Changed
// so they only appear in the request body when explicitly passed (RESEARCH
// Pattern 3 gating).
//
// NOTE: Uses DoCreate (not the owner-scoped variant used by teams) —
// organizations are not owner-scoped (RESEARCH Anti-Patterns to Avoid).
func newCreateCmd() *cobra.Command {
	var name, externalID, parentID string

	c := &cobra.Command{
		Use:         "create",
		Short:       "Create a new organization",
		Annotations: map[string]string{"mutating": "true"}, // CF-12-16
		Example: `  # Create an organization with a name only
  revenium organizations create --name "Acme Corporation"

  # Create with an external system identifier
  revenium organizations create --name "Acme Corporation" --external-id ext-123

  # Create as a child of an existing organization
  revenium organizations create --name "Acme NA" --parent-id parent-org-456`,
		RunE: func(c *cobra.Command, args []string) error {
			// --name is MarkFlagRequired below, so it is always populated.
			body := map[string]interface{}{
				"name": name,
			}
			if c.Flags().Changed("external-id") {
				body["externalId"] = externalID
			}
			if c.Flags().Changed("parent-id") {
				body["parentId"] = parentID
			}

			if cmd.DryRun() { // CF-12-17 — gate fires BEFORE HTTP
				return dryrun.Render(cmd.Output, "create", "organization", "/v2/api/organizations", body)
			}

			var result map[string]interface{}
			// CF-12-20: DoCreate auto-injects tenantId from client config.
			// Do NOT use the owner-scoped variant — organizations are not owner-scoped.
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/organizations", body, &result); err != nil {
				return err
			}
			return renderOrg(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Organization name")
	c.Flags().StringVar(&externalID, "external-id", "", "External system identifier")
	c.Flags().StringVar(&parentID, "parent-id", "", "Parent organization ID")
	_ = c.MarkFlagRequired("name") // RESEARCH D-05 LOCKED: only --name is required

	return c
}
