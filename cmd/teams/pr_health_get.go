package teams

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// prHealthEffectiveValuesCaption is the standing provenance caption printed
// above the rendered table in non-JSON mode (D-31-01).
//
// Source: the dev platform document's GET /v2/api/teams/{id}/settings/pr-health
// operation description, which states the endpoint returns effective values,
// with the API's own defaults already applied when a team has never configured
// them. The response carries no provenance field, so the CLI cannot tell a
// configured value from a server default.
//
// Two things the CLI therefore deliberately does NOT do:
//   - It declares no client-side table of the schema's `example` threshold
//     values. Hardcoding a prose-documented default client-side is D-22-04's
//     recorded v1.3 anti-pattern (see cmd/skills/skills.go).
//   - It labels no individual row as a default. A team that deliberately
//     configured those exact numbers would be told it never configured them.
//
// A single standing caption over the whole table says the true thing — the
// values are effective — without inventing a per-row fact the API refuses to
// supply.
const prHealthEffectiveValuesCaption = "Effective values — the API applies its own defaults when a team has never configured these."

func newPrHealthGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <team-id>",
		Short: "View PR health thresholds for a team",
		Long: `View the effective PR health thresholds for a team.

The API returns effective values with its own defaults already applied when a
team has never configured these thresholds. The response carries no provenance,
so the CLI cannot distinguish a value a team configured from a default the
server supplied, and it deliberately does not guess: no row is labelled as a
default and no client-side table of default values exists.`,
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # View the effective PR health thresholds
  revenium teams pr-health get team-123

  # View as JSON
  revenium teams pr-health get team-123 --json

  # Show only the values
  revenium teams pr-health get team-123 --fields Value`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/teams/%s/settings/pr-health", url.PathEscape(args[0]))

			var settings map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &settings); err != nil {
				return err
			}

			// Printed BEFORE the render so it reads as a heading rather than a
			// footnote, and gated on non-JSON mode so it can never land inside
			// a --json document and corrupt a machine consumer's parse (T-31-02).
			//
			// Written to the formatter's OWN writer, not to c.OutOrStdout():
			// the formatter substitutes io.Discard in quiet non-JSON mode
			// (internal/output/output.go), so quiet suppression is a property
			// of the formatter rather than of this call site, and a line
			// written past the formatter escapes it. That matters here more
			// than most: the caption exists only to QUALIFY the table, and
			// RenderTable returns early under --quiet — so a caption that
			// survived would qualify something that is not there (code review
			// WR-03, IN-06). This is the writer
			// cmd/guardrails/org_unit_group_preview.go already uses.
			if !cmd.Output.IsJSON() {
				fmt.Fprintln(cmd.Output.Writer(), prHealthEffectiveValuesCaption)
			}

			return renderPrHealthSettings(settings)
		},
	}
}
