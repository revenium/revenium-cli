package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newEconomicsGetCmd builds `revenium jobs types economics get <type>` — the
// read half of JOBS-11.
//
// GET /v2/api/jobs/types/{type}/economics returns a JobTypeEconomicsResource.
// It is decoded into a map, never a typed struct: re-encoding a struct for
// --json would drop any server field the struct does not know about, breaking
// the get/edit/set round-trip (D-27-03) the day the spec grows a key.
func newEconomicsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <type>",
		Short: "View a job type's declared economics contract",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Example: `  # View the declared economics contract for a job type
  revenium jobs types economics get acme-review

  # As JSON for scripting
  revenium jobs types economics get acme-review --json`,
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
			path := fmt.Sprintf("/v2/api/jobs/types/%s/economics", url.PathEscape(args[0]))

			var economics map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &economics); err != nil {
				return err
			}
			return renderEconomics(economics)
		},
	}
}
