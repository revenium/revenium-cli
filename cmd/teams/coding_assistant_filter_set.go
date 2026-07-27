package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newCodingAssistantFilterSetCmd returns the `set <id>` sub-command for
// coding-assistant-filter settings. PUTs directly to the settings sub-resource
// via cmd.APIClient.Do — no GET-merge-style update helper is used, matching
// prompt-capture set.
//
// Assumption A4 (documented decision, RESEARCH RES-08 / Assumptions Log A4):
// the CodingAssistantFilterSettingsResource schema mixes the live field
// `apiRateProviders` with three fields described in schema prose as
// deprecated legacy fields (`enabled`, `defaultProviders`, `allowUserOverride`).
// Only `--api-rate-providers` is exposed as writable here; the deprecated-in-
// prose fields are intentionally NOT exposed as flags. If full back-compat
// coverage of all seven fields is later desired, that's a defensible
// alternative — flag for the Phase 8 drift report if revisited.
func newCodingAssistantFilterSetCmd() *cobra.Command {
	var apiRateProviders []string

	c := &cobra.Command{
		Use:   "set <id>",
		Short: "Update coding-assistant filter settings for a team",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Set the allowed API rate providers
  revenium teams coding-assistant-filter set team-123 --api-rate-providers ClaudeCode,CursorIde

  # Dry run
  revenium teams coding-assistant-filter set team-123 --api-rate-providers ClaudeCode --dry-run`,
		Annotations: map[string]string{"mutating": "true"},
		RunE: func(c *cobra.Command, args []string) error {
			body := make(map[string]interface{})

			if c.Flags().Changed("api-rate-providers") {
				body["apiRateProviders"] = apiRateProviders
			}

			if len(body) == 0 {
				return fmt.Errorf("no fields specified to update")
			}

			path := fmt.Sprintf("/v2/api/teams/%s/settings/coding-assistant-filter", url.PathEscape(args[0]))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "coding-assistant filter settings", path, body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}
			return renderCodingAssistantFilterSettings(result)
		},
	}

	c.Flags().StringSliceVar(&apiRateProviders, "api-rate-providers", nil, "Comma-separated or repeated list of allowed API rate providers (ClaudeCode, ClaudeCowork, CursorIde, GeminiCli, CodexCli, GithubCopilot)")

	return c
}
