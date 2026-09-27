package jobs

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// init registers the outcome-metrics subcommand on the parent jobs.Cmd,
// following the established multi-init-per-package pattern (see outcome.go).
//
// This file carries its own initializer, unlike facts.go: Cmd is a
// package-level variable that exists before any initializer in this package
// runs, so AddCommand here is safe regardless of the lexical order Go visits
// these files in. facts.go must NOT have one because it hangs off typesCmd,
// which is itself built by an initializer.
func init() {
	Cmd.AddCommand(newOutcomeMetricsCmd())
}

// newOutcomeMetricsCmd builds `revenium jobs outcome-metrics <agenticJobId>`
// — JOBS-16.
//
// WHY THE HYPHEN, AND WHY THIS IS NOT `jobs outcome metrics` (D-28-02).
// The shipped `jobs outcome <agenticJobId>` is an exact-one-argument LEAF: it
// declares cobra.ExactArgs(1) and a RunE that reports an outcome. Nesting a
// `metrics` child under it would convert a released command into a parent,
// changing the behaviour operators already depend on — a bare
// `jobs outcome <id>` would begin resolving its first positional as a
// subcommand name. It would also strand outcome-history and outcome-update,
// the two shipped members of this same hyphenated family, as inconsistent
// siblings. The hyphen maps the URL's /outcome/metrics segment pair exactly.
// A future reader who finds the nesting tidier must read this paragraph
// first: the tidiness costs a behaviour change on a released command.
//
// The endpoint is append-only. Nothing on any cached spec reads an appended
// metric back — /outcome/history returns outcome revisions and /roi returns
// aggregates — so the help text below promises no read-back and the success
// line reports what was sent rather than what the platform now holds.
//
// Input arrives as a whole JSON array through --file rather than as one
// entry's worth of scalar flags, for the same reason facts append does: a
// backfill is tens of entries, and a flag-per-field command would be a shell
// loop issuing one irreversible POST per entry.
//
// --dry-run is honoured. The `mutating` annotation below is published to
// automation by cmd/schema.go and the README promises --dry-run previews any
// mutation without executing it, so declaring the annotation without reading
// the flag would perform an irreversible POST on a command an operator asked
// to preview. Both halves, never one.
func newOutcomeMetricsCmd() *cobra.Command {
	// Scoped to the constructor, not to the package. `facts append` binds its
	// own --file the same way, and a package-level binding would be reachable
	// by every command in cmd/jobs — two commands sharing one --file value is
	// a live bug the day both exist.
	var file string

	c := &cobra.Command{
		Use:   "outcome-metrics <agenticJobId>",
		Short: "Append late per-job outcome metrics to an existing job",
		Long: `Append late per-job outcome metrics to an existing job from a JSON file or stdin.

The --file argument is a JSON array of entries. Every entry in the array is
sent, in the order the array gives them; nothing is merged, deduplicated or
dropped, because two identical measurements are two measurements.

Each entry carries recordedAt, the moment the metric was observed, which is
what makes a late append meaningful long after the job itself finished.

Appended metrics are permanent — the API offers no amendment and no deletion —
so every entry is checked locally before any request is issued, and --dry-run
reports the request that would be sent without sending it. The platform
exposes no way to read an appended metric back, so a successful run reports
what this command sent, not what the server now holds.`,
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Annotations: map[string]string{"mutating": "true"},                   // D-16
		Example: `  # Append the outcome metrics held in a file
  revenium jobs outcome-metrics loan-app-12345 --file metrics.json

  # Append from a pipe
  cat metrics.json | revenium jobs outcome-metrics loan-app-12345 --file -

  # Preview the request without issuing it
  revenium jobs outcome-metrics loan-app-12345 --file metrics.json --dry-run`,
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
			//
			// One reader serves both append commands. OutcomeMetricEntry is
			// PeriodFactEntry minus the period window and the dimension pair,
			// plus recordedAt — the array framing, the five source-naming
			// refusals and the map-not-struct decode are identical, and a
			// second copy of them would drift.
			entries, err := readFactEntries(file, c.InOrStdin())
			if err != nil {
				return err
			}

			// D-28-06: one validator, both entry kinds. The provenance enum
			// on OutcomeMetricEntry is byte-identical to the fact entry's and
			// the quality_rate sentence is the same sentence, so the same
			// indexed refusals apply here unchanged.
			//
			// The gate sits above the path construction, which is what makes
			// "no request is made" structural rather than merely true — when
			// an entry is rejected there is no request object in existence to
			// issue.
			if err := validateFactEntries(entries); err != nil {
				return err
			}

			// D-25 / CF-13: defensive escaping because <agenticJobId> is
			// user-supplied. Computed ONCE, above the dry-run check, so the
			// preview and the real request cannot diverge.
			path := fmt.Sprintf("/v2/api/jobs/%s/outcome/metrics", url.PathEscape(args[0]))

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
				// that. Under --json there is no such formatting problem, so
				// the entries themselves go through.
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
				return dryrun.Render(cmd.Output, "append", "job outcome metrics", path, body)
			}

			// The fifth argument is nil, NOT a decode target. This endpoint
			// answers 201 with no content block at all, and Client.Do decodes
			// the response body whenever that argument is non-nil — so
			// passing one turns every successful append into "failed to
			// decode response: EOF". baselines_append.go passes &appended
			// because ITS endpoint declares a 201 body; this one does not.
			if err := cmd.APIClient.Do(c.Context(), "POST", path, entries, nil); err != nil {
				return err
			}

			// The server returned nothing, so the only truthful report is
			// what was sent. Echo the accepted array verbatim under --json
			// rather than inventing a server-shaped resource, which would
			// report a metric the platform never confirmed.
			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(entries)
			}
			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Appended %d outcome metric%s to job %s.\n",
					len(entries), pluralSuffix(len(entries)), args[0])
			}
			return nil
		},
	}

	// The usage string carries no parenthesised required marker: cmd/help.go
	// appends " (required)" itself for any flag carrying cobra's required
	// annotation, so spelling it out here renders it twice.
	c.Flags().StringVar(&file, "file", "", "Path to a JSON array of outcome metric entries, or - to read from stdin")
	_ = c.MarkFlagRequired("file")

	return c
}
