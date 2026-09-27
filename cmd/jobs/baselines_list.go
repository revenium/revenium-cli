package jobs

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// newBaselinesListCmd builds `revenium jobs types baselines list <type>` —
// JOBS-13.
//
// GET /v2/api/jobs/types/{type}/baselines returns a bare
// array<BaselineResource>. Each element is decoded into a map, never a typed
// struct: re-encoding a struct for --json would drop any server field the
// struct does not know about, the day the spec grows a key.
func newBaselinesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <type>",
		Short: "List a job type's immutable pre-AI baseline versions, newest first",
		Long: "List a job type's immutable pre-AI baseline versions, newest first.\n\n" +
			"Row 1 is the version currently in force. Every version ever declared\n" +
			"remains listed here with the values it was declared with.",
		Args: cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Example: `  # List the baseline versions declared for a job type
  revenium jobs types baselines list acme-review

  # As JSON for scripting — same order as the table
  revenium jobs types baselines list acme-review --json`,
		RunE: func(c *cobra.Command, args []string) error {
			// D-27-10 / SC5: refuse before anything reaches the wire. This
			// must stay the first statement — the endpoint requires teamId
			// and internal/api appends it only when non-empty, so without
			// the guard the request is issued team-less and the failure is
			// silent.
			if err := requireTeam(); err != nil {
				return err
			}

			// D-25 / CF-13: defensive PathEscape because <type> is
			// user-supplied. The cobra arg validator already rejects ?, &,
			// #, %, ../, ..\ and control chars; PathEscape covers what the
			// validator allows through — most importantly "/", which
			// appears as %2F.
			path := fmt.Sprintf("/v2/api/jobs/types/%s/baselines", url.PathEscape(args[0]))

			// The endpoint returns a bare array, not a Spring HATEOAS page,
			// so the auto-pagination helper does not fit this shape — the
			// same situation cmd/jobs/types.go already documents for its
			// []string decode.
			var baselines []map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &baselines); err != nil {
				return err
			}

			// Empty-state branching per Phase 12 D-09 (cmd/jobs/list.go:26-32).
			// The typed empty slice is load-bearing: a nil Go slice marshals
			// to `null`, which is a different value to a consumer than `[]`.
			if len(baselines) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]map[string]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No baseline versions declared for this job type.")
				return nil
			}

			// Defence in depth — NOT gap-filling, and not redundant.
			//
			// Correction C4: the spec's 200 description does say, verbatim,
			// "Baseline versions returned newest first." A future reader who
			// finds that sentence must not delete this sort as duplicated
			// work. Sorting here is what makes JOBS-13's "row 1 is the
			// version in force" a property of OUR output rather than a
			// belief about a server that only a stub could ever confirm —
			// which is the shape Phase 26 found green five times under
			// mutations it should have caught.
			//
			// SliceStable, not Slice: two rows carrying an equal `version`
			// keep the server's relative order rather than being reordered
			// arbitrarily, and neither is merged nor dropped.
			//
			// json.Unmarshal decodes every JSON number as float64, which is
			// what output.FloatVal reads.
			sort.SliceStable(baselines, func(i, j int) bool {
				return output.FloatVal(baselines[i], "version") > output.FloatVal(baselines[j], "version")
			})

			// The SORTED slice is the third argument too, so the table and
			// --json are produced from one ordering and cannot disagree
			// about which version is first.
			return cmd.Output.Render(baselinesTableDef, toBaselineRows(baselines), baselines)
		},
	}
}
