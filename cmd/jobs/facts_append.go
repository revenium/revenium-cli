package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newFactsAppendCmd builds `revenium jobs types facts append <type>` —
// JOBS-15.
//
// The verb is `append`, matching the spec's own summary "Append declared
// PERIOD outcome facts". `create` is the repo's general convention for a POST,
// but it reads as "make a new thing" and says nothing about what happens to
// what was there before — which is exactly the impression the append-only
// story exists to avoid.
//
// Input arrives as a whole JSON array through --file, rather than as one
// entry's worth of scalar flags: PeriodFactEntry carries ten properties and a
// month of facts is tens of entries, so a flag-per-field command would be a
// shell loop issuing one irreversible POST per entry.
//
// --dry-run is honoured, using the same shape as every other mutating command
// in cmd/. The `mutating` annotation below is published to automation by
// cmd/schema.go and the README promises --dry-run previews any mutation
// without executing it, so declaring the annotation without reading the flag
// would perform an irreversible POST on a command an operator asked to
// preview. Both halves, never one.
func newFactsAppendCmd() *cobra.Command {
	// Scoped to the constructor rather than to the package, as
	// economics_set.go and models/pricing_bulk_save.go both do. The command is
	// constructed exactly once at package initialization, so this outlives the
	// constructor for as long as the command exists; a package-level binding
	// would additionally be reachable by every other command in cmd/jobs and
	// invite two commands sharing one --file value.
	var file string

	c := &cobra.Command{
		Use:   "append <type>",
		Short: "Append declared period outcome facts to a job type from a JSON file or stdin",
		Long: `Append declared period outcome facts to a job type from a JSON file or stdin.

The --file argument is a JSON array of entries. Every entry in the array is
sent, in the order the array gives them; nothing is merged, deduplicated or
dropped, because two identical measurements are two measurements.

Appended facts are permanent — the API offers no amendment and no deletion —
so every entry is checked locally before any request is issued, and --dry-run
reports the request that would be sent without sending it. The platform
exposes no way to read an appended fact back, so a successful run reports what
this command sent, not what the server now holds.`,
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Annotations: map[string]string{"mutating": "true"},                   // D-16
		Example: `  # Append the period facts held in a file
  revenium jobs types facts append acme-review --file facts.json

  # Append from a pipe
  cat facts.json | revenium jobs types facts append acme-review --file -

  # Preview the request without issuing it
  revenium jobs types facts append acme-review --file facts.json --dry-run`,
		RunE: func(c *cobra.Command, args []string) error {
			// D-27-10 / D-28-12: refuse before anything reaches the wire.
			// This must stay the first statement — the endpoint declares
			// teamId required and internal/api appends it only when
			// non-empty, so without the guard the request is issued
			// team-less and the operator reads a confusing 4xx.
			if err := requireTeam(); err != nil {
				return err
			}

			// The reader comes from cobra rather than from os.Stdin directly
			// so the `-` path is testable without process-level plumbing.
			entries, err := readFactEntries(file, c.InOrStdin())
			if err != nil {
				return err
			}

			// SC2: every failure in the file, each named by its array index,
			// reported BEFORE the path exists. Placing the gate above the
			// path construction is what makes "no request is made" structural
			// rather than merely true — when an entry is rejected there is no
			// request object in existence to issue.
			if err := validateFactEntries(entries); err != nil {
				return err
			}

			// D-25 / CF-13: defensive PathEscape because <type> is
			// user-supplied. Computed ONCE, above the dry-run check, so the
			// preview and the real request cannot diverge.
			path := fmt.Sprintf("/v2/api/jobs/types/%s/facts", url.PathEscape(args[0]))

			// --dry-run wins over everything, including --yes: it is checked
			// immediately after the path is resolved and returns before any
			// request is issued. The assertion that matters is the POST
			// counter being zero, not that the command exited 0.
			if cmd.DryRun() { // D-17
				// dryrun.Render's table path prints the body with %v, which
				// for a slice of maps is Go map syntax in non-deterministic
				// key order — not a preview anyone reads for a 200-entry
				// file. D-28-04 asks for the method, the path and the
				// parsed entry count, so the table path gets exactly
				// that. Under --json there is no such formatting problem and
				// no earlier copy of the array, so the entries themselves go
				// through.
				// The parenthetical names the two checks validateFactEntries
				// actually applies — provenance enum membership and the
				// quality_rate bound — rather than claiming the entries were
				// "validated". That word covered required-ness and shape to
				// every operator who read it, and covers neither: an entry with
				// no key, no value, an undeclared metric, or no fields at all
				// passes this path unrefused. Overstating it here is worse than
				// saying nothing, because CONTEXT.md tells agent consumers to
				// use --dry-run to validate generated commands, and because an
				// accepted entry is permanent and cannot be read back.
				var body interface{} = fmt.Sprintf("POST %s (parsed; provenance and quality_rate checked)", pluralEntries(len(entries)))
				if cmd.Output.IsJSON() {
					body = entries
				}
				return dryrun.Render(cmd.Output, "append", "job type period facts", path, body)
			}

			// The fifth argument is nil, NOT a decode target. This endpoint
			// answers 201 with no content block at all, and Client.Do decodes
			// the response body whenever that argument is non-nil — so
			// passing one turns every successful append into "failed to
			// decode response: EOF". baselines_append.go passes &appended
			// because ITS endpoint declares a 201 body; do not "fix" this
			// call into that shape.
			if err := cmd.APIClient.Do(c.Context(), "POST", path, entries, nil); err != nil {
				return err
			}

			// The server returned nothing, so the only truthful report is
			// what was sent. Echo the accepted array verbatim under --json
			// rather than inventing a server-shaped resource, which would
			// report a fact the platform never confirmed.
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(entries)
			}
			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Appended %d period fact%s to job type %s.\n",
					len(entries), pluralSuffix(len(entries)), args[0])
			}
			return nil
		},
	}

	// The usage string carries no parenthesised required marker: cmd/help.go
	// appends " (required)" itself for any flag carrying cobra's required
	// annotation, so spelling it out here renders it twice.
	c.Flags().StringVar(&file, "file", "", "Path to a JSON array of period fact entries, or - to read from stdin")
	_ = c.MarkFlagRequired("file")

	return c
}
