package skills

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() { Cmd.AddCommand(newListCmd()) }

// listTableDef defines the skill-usage table layout (GAP-01, D-22-06).
// Columns render eight of the 13 verified EntityModelSkillUsageResource_Read
// properties. resourceType is omitted (constant "skill") and _links is
// omitted (HATEOAS plumbing); --json remains the full-fidelity path.
// StatusColumn is -1 because none of the 13 properties is a status.
var listTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Kind", "Source", "Origin", "Calls", "Traces", "Cost"},
	StatusColumn: -1,
}

// toListRows converts a slice of skill-usage maps to table row strings.
func toListRows(skills []map[string]interface{}) [][]string {
	rows := make([][]string, len(skills))
	for i, s := range skills {
		rows[i] = []string{
			str(s, "id"),
			str(s, "name"),
			str(s, "kind"),
			str(s, "source"),
			str(s, "originCategory"),
			formatNumber(floatVal(s, "callCount")),
			formatNumber(floatVal(s, "traceCount")),
			formatCost(floatVal(s, "totalCost")),
		}
	}
	return rows
}

// newListCmd returns the `revenium skills list` subcommand
// (GET /v2/api/skills, operation listSkills, GAP-01). teamId is required by
// the spec and auto-injected by cmd.APIClient.DoList from resolved config —
// no --team-id flag is added here (D-22-03), since the global override flag
// already covers per-invocation scoping and a package-local flag would be a
// second, conflicting source of truth.
func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List skills with their usage and attributed cost",
		Args:  cobra.NoArgs,
		Example: `  # List skills, most expensive first
  revenium skills list

  # List skill usage over the last 7 days
  revenium skills list --period SEVEN_DAYS

  # List skills as JSON
  revenium skills list --json`,
		RunE: func(c *cobra.Command, args []string) error {
			var skills []map[string]interface{}
			path := buildSkillsPath("/v2/api/skills")
			if err := cmd.APIClient.DoList(c.Context(), path, cmd.ListOptsFromFlags(c), &skills); err != nil {
				return err
			}
			if len(skills) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No skills found.")
				return nil
			}
			return cmd.Output.Render(listTableDef, toListRows(skills), skills)
		},
	}

	cmd.AddListFlags(c)
	return c
}
