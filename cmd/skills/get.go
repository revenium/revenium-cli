package skills

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newGetCmd()) }

// getTableDef defines the skill-detail table layout (GAP-02, D-22-06).
// Columns render 12 of the 13 verified SkillUsageResource_Read properties.
// resourceType is omitted because it is the constant "skill" and carries no
// information for a reader who already typed `skills get`; _links, when the
// server sends it, is HATEOAS plumbing. Neither is lost: --json remains the
// full-fidelity path and emits the response object unmodified.
//
// StatusColumn is -1 because none of the 13 properties is a status field.
var getTableDef = output.TableDef{
	Headers: []string{
		"ID", "Name", "Kind", "Source", "Origin", "Plugin", "Marketplace",
		"Calls", "Traces", "Cost", "First Seen", "Last Seen",
	},
	StatusColumn: -1,
}

// renderSkill renders a single skill's usage detail as a one-row table, or as
// the raw response object under --json.
//
// The two timestamps are rendered by str — raw ISO-8601, no reformatting.
// str also returns "" for a nil value, which is exactly what the seven
// nullable properties (kind, source, originCategory, pluginName,
// marketplaceName, firstSeen, lastSeen) need: an empty cell rather than the
// "<nil>" that fmt.Sprint would otherwise produce.
func renderSkill(skill map[string]interface{}) error {
	rows := [][]string{{
		str(skill, "id"),
		str(skill, "name"),
		str(skill, "kind"),
		str(skill, "source"),
		str(skill, "originCategory"),
		str(skill, "pluginName"),
		str(skill, "marketplaceName"),
		formatNumber(floatVal(skill, "callCount")),
		formatNumber(floatVal(skill, "traceCount")),
		formatCost(floatVal(skill, "totalCost")),
		str(skill, "firstSeen"),
		str(skill, "lastSeen"),
	}}
	return cmd.Output.Render(getTableDef, rows, skill)
}

// newGetCmd returns the `revenium skills get <skillId>` subcommand
// (GET /v2/api/skills/{skillId}, operation getSkillDetail, GAP-02).
//
// The response is a flat SkillUsageResource_Read object, not a HATEOAS
// envelope, so this calls Do rather than DoList.
//
// T-22-02 mitigation: cmd.ValidResourceID rejects path traversal and control
// characters at argument-validation time, before RunE ever runs, so a hostile
// id never reaches the transport; url.PathEscape then encodes whatever
// survives. The ordering is the mitigation. This is the same guard
// cmd/squads/get.go carries for T-2-01.
//
// No local --team-id flag (D-22-03; teamId is auto-injected by cmd.APIClient
// and the global override covers retargeting) and no --sort flag (D-22-05).
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <skillId>",
		Short: "Get a skill's usage and attributed cost by Revenium skill id",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a skill's usage detail
  revenium skills get JMwX9g4

  # Get a skill's usage detail over the last 7 days
  revenium skills get JMwX9g4 --period SEVEN_DAYS

  # Get a skill's usage detail as JSON
  revenium skills get JMwX9g4 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// The literal endpoint path stays the first argument to
			// buildSkillsPath so the Phase 20 extractor resolves this call
			// site to /v2/api/skills/{param}.
			path := buildSkillsPath(fmt.Sprintf("/v2/api/skills/%s", url.PathEscape(args[0])))
			var skill map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &skill); err != nil {
				return err
			}
			return renderSkill(skill)
		},
	}
}
