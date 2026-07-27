package anomalies

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// clearPath is the literal destructive DELETE-all path for anomaly detection
// rules: DELETE /v2/api/sources/ai/anomaly/clear (clear_all_ai_anomalies).
// This clears ALL rules for the team, not one — see the full-destructive
// confirmation treatment below (T-05-08).
const clearPath = "/v2/api/sources/ai/anomaly/clear"

func newClearCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "clear",
		Short:       "Delete ALL anomaly detection rules for this team",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.NoArgs,
		Example: `  # Clear all anomaly detection rules (with confirmation)
  revenium anomalies clear

  # Clear without confirmation
  revenium anomalies clear --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "clear", "all anomalies", clearPath, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			// This action targets ALL anomaly detection rules for the team, not a
			// single resource, so there is no id to pass. Calling ConfirmDelete
			// with an empty id (and a descriptive resourceType label) is
			// intentional here; the resulting prompt still reads naturally
			// ("Delete ALL anomaly detection rules for this team ? [y/N]"),
			// with a harmless extra space before the "?" from the empty id.
			ok, err := resource.ConfirmDelete("ALL anomaly detection rules for this team", "", yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := cmd.APIClient.Do(c.Context(), "DELETE", clearPath, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintln(c.OutOrStdout(), "Cleared all anomaly detection rules for this team.")
			}
			return nil
		},
	}

	return c
}
