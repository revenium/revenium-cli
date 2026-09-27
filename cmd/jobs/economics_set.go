package jobs

import (
	"bufio"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	rerrors "github.com/revenium/revenium-cli/internal/errors"
)

// newEconomicsSetCmd builds `revenium jobs types economics set <type>` —
// JOBS-12's write path.
//
// It composes the four pure stages in economics_doc.go and puts a fail-closed
// guard in front of the PUT:
//
//	requireTeam -> readEconomicsInput -> stripReadOnlyKeys -> validateEconomicsDocument
//	           -> preflight GET -> diffEconomics -> the precedence ladder -> PUT
//
// Every stage before the ladder can refuse, and the ladder itself can refuse;
// only the far end writes.
//
// The scaffolding (the --file flag, MarkFlagRequired, the "mutating"
// annotation, the destructive-full-replace Long block) is copied from
// cmd/models/pricing_bulk_save.go. Its SAFETY POSTURE is deliberately not:
// that command issues no preflight GET and asks for no confirmation, so it
// cannot tell the operator what a replace would cost. This one can, and
// therefore must.
func newEconomicsSetCmd() *cobra.Command {
	var file string

	c := &cobra.Command{
		Use:   "set <type>",
		Short: "Create or replace a job type's declared economics contract from a file or stdin",
		Long: `Create or replace a job type's declared economics contract from a file or stdin.

WARNING: this replaces the WHOLE economics document. Any metric, dimension or
allowed value that is not present in the supplied JSON is removed, and the
monetization rule is deleted if the document omits it. There is no merge — that
is what makes deleting a metric possible at all.

Use --dry-run to preview the resolved request body and exactly what the replace
would remove, without writing anything. A replace that would remove something
the current contract declares requires --yes to proceed.

The read-only keys jobType and currentBaseline are stripped from the supplied
document and reported on stderr, so a document produced by ` + "`economics get --json`" + `
can be edited and fed straight back. Baselines are append-only and are never
changed by this command; edit currentBaseline all you like and nothing happens.`,
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID), // CF-17
		Annotations: map[string]string{"mutating": "true"},                   // D-16
		Example: `  # The intended editing loop, end to end
  revenium jobs types economics get acme-review --json > economics.json
  $EDITOR economics.json
  revenium jobs types economics set acme-review --file economics.json

  # Straight from a pipe. Note that --file - ALWAYS requires --yes for a
  # destructive replace: stdin was consumed by the document, so there is no
  # terminal left to confirm on.
  cat economics.json | revenium jobs types economics set acme-review --file - --yes

  # Preview the resolved body and what would be lost, writing nothing
  revenium jobs types economics set acme-review --file economics.json --dry-run

  # Proceed with a replace that removes a declared metric
  revenium jobs types economics set acme-review --file economics.json --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			// D-27-10 / SC5: refuse before anything reaches the wire, and
			// before the file is even read. This must stay the first
			// statement — the endpoint requires teamId and internal/api
			// appends it only when non-empty, so without the guard the
			// request is issued team-less and the failure is silent.
			if err := requireTeam(); err != nil {
				return err
			}

			// The reader comes from cobra rather than from os.Stdin directly
			// so the `-` path is testable without process-level plumbing.
			doc, err := readEconomicsInput(file, c.InOrStdin())
			if err != nil {
				return err
			}

			// D-27-03. The note is load-bearing, not decoration: an operator
			// who edited currentBaseline must not be told the write succeeded
			// while believing the baseline moved, because baselines are
			// append-only and that belief would be actively wrong.
			//
			// It goes to stderr, never stdout, so --json stdout stays
			// machine-clean, and it is gated on the repo's quiet convention.
			body, dropped := stripReadOnlyKeys(doc)
			if len(dropped) > 0 && !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.ErrOrStderr(),
					"Note: ignoring read-only key(s) %s — they are not part of the economics contract. "+
						"Baselines are append-only and are not changed by `economics set`.\n",
					strings.Join(dropped, ", "))
			}

			// Returned unchanged: the message already names the JSON path
			// (including its array index) and the full list of legal values.
			if err := validateEconomicsDocument(body); err != nil {
				return err
			}

			// D-25 / CF-13: defensive PathEscape because <type> is
			// user-supplied. Computed ONCE and used for the preflight GET,
			// the dry-run render and the PUT, so the three can never diverge.
			path := fmt.Sprintf("/v2/api/jobs/types/%s/economics", url.PathEscape(args[0]))

			// The preflight GET. It is issued HERE rather than from a helper
			// so that `path` is a local at the call site: tools/coverage-audit
			// resolves every Client.Do call site to an API path by tracing the
			// path argument within its enclosing function, and a path arriving
			// as a parameter resolves to nothing and fails
			// TestExtractRealTree. The classification is still a named
			// function, because that is where the reasoning belongs.
			//
			// A 404 means no contract has been declared yet. The spec
			// documents PUT as an upsert, so creating a contract via `set` on
			// a job type that has none must work, and there is by definition
			// nothing to lose — diffEconomics already classifies an empty
			// current contract as non-destructive.
			//
			// Anything else — 401, 403, 409, 422, 429, 500, or a transport
			// failure — returns unchanged and issues NO PUT. We cannot know
			// what would be lost, and proceeding anyway would convert the
			// destructive guard into a coin flip at exactly the moment the API
			// is unhealthy. Blocking costs the operator a retry; degrading to
			// a warning costs them a contract.
			var current map[string]interface{}
			if getErr := cmd.APIClient.Do(c.Context(), "GET", path, nil, &current); getErr != nil {
				if !isEconomicsNotFound(getErr) {
					return getErr
				}
				current = nil
			}
			// A 404, or a 200 carrying a null or empty body, both leave a nil
			// map. The diff stages read a nil map correctly; normalising here
			// keeps every later reader free of the distinction.
			if current == nil {
				current = map[string]interface{}{}
			}

			diff := diffEconomics(current, body)

			// RUNG 1 — dry run always wins, and never writes.
			//
			// It comes first deliberately: --dry-run --yes must still write
			// nothing (D-27-06), and a destructive diff must not be able to
			// turn a preview into a refusal. The diff renders either way, so
			// the operator learns from the preview whether the real run will
			// need --yes.
			if cmd.DryRun() {
				renderDestructiveDiff(c.ErrOrStderr(), diff)

				// dryrun.Render's non-JSON path prints the body with an
				// unformatted verb, which for a nested economics document is
				// Go map syntax with non-deterministic key order (RESEARCH
				// Pitfall 8). Print the resolved body as indented JSON here
				// and hand the helper a nil body, so the document is rendered
				// once and only in the readable form. Requiring --json to read
				// a human-facing affordance would defeat its purpose, and a
				// test asserting the output merely contains "metrics" would
				// pass against the map syntax and prove nothing.
				if !cmd.Output.IsJSON() {
					pretty, err := json.MarshalIndent(body, "", "  ")
					if err != nil {
						return fmt.Errorf("failed to render the resolved economics document: %w", err)
					}
					// cmd.Output.Writer(), not c.OutOrStdout(): it is the
					// same stdout in production but is io.Discard under
					// --quiet, which is also where dryrun.Render writes. Split
					// across two writers, --quiet would print the body and
					// swallow the footer.
					fmt.Fprintf(cmd.Output.Writer(), "Resolved request body:\n%s\n", pretty)

					// nil, not body. dryrun.Render prints `  Body: %v`
					// whenever body != nil, so passing it here would print
					// the document a SECOND time in exactly the form the
					// block above exists to avoid — Go map syntax, with
					// non-deterministic key order and e-notation for any
					// number at or above 1e6. The shared helper still
					// supplies the header, the path and the footer, which is
					// all this path wants from it.
					return dryrun.Render(cmd.Output, "replace", "job type economics", path, nil)
				}

				// Under --json there is no earlier copy, so the body must
				// reach the helper: it is the whole payload of the structured
				// preview.
				return dryrun.Render(cmd.Output, "replace", "job type economics", path, body)
			}

			// RUNGS 2-5. Only a nil return here reaches the PUT.
			if err := confirmDestructiveReplace(c, file, diff); err != nil {
				return err
			}

			// The PUT is a full replace; no read-modify-write merge is
			// performed, deliberately. Merging would remove the only way to
			// delete a metric and would leave array semantics ambiguous
			// between replace and union.
			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "PUT", path, body, &result); err != nil {
				return err
			}

			// Echo the contract back as it now stands. This is what makes
			// SC2's "reading it back returns what was written" observable from
			// the command itself rather than only from a follow-up `get`.
			return renderEconomics(result)
		},
	}

	// Exactly one local flag. --yes and --dry-run are PERSISTENT ROOT flags
	// (cmd/root.go), read through cmd.YesMode() and cmd.DryRun(); declaring
	// either here would shadow the root flag and silently break both
	// accessors, so this command declares no bool flag at all.
	//
	// The usage string deliberately does NOT end in "(required)": cmd/help.go
	// appends that marker itself for any flag carrying the required
	// annotation, so spelling it out here renders as "(required) (required)".
	c.Flags().StringVar(&file, "file", "",
		"Path to a JobTypeEconomics JSON document, or - to read it from stdin")
	_ = c.MarkFlagRequired("file")

	return c
}

// isEconomicsNotFound reports whether a preflight failure was a 404 — the one
// failure that is not a failure, because it means the contract does not exist
// yet and `set` is creating rather than replacing.
//
// The status is read by unwrapping to the API error type, never by matching on
// message text. mapHTTPError's 404 wording ("Resource not found.") is a
// presentation detail that can be reworded without notice, and a guard keyed on
// prose is a guard that silently stops guarding — it would start treating every
// 500 as a create, which is the precise failure the preflight exists to
// prevent.
func isEconomicsNotFound(err error) bool {
	var apiErr *rerrors.APIError
	return stderrors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// renderDestructiveDiff writes the loss summary to w. Callers always pass
// c.ErrOrStderr(), so the summary reaches the operator in every mode —
// including --dry-run --json, where stdout must stay machine-parseable while
// the operator still learns what would be lost.
//
// The summary names each lost entry by key and by collection. A count alone is
// not actionable: "3 items would be lost" tells an operator to go and diff two
// files by hand, which is the work this line exists to do for them.
func renderDestructiveDiff(w io.Writer, d economicsDiff) {
	if !d.IsDestructive() {
		fmt.Fprintf(w, "This replace is not destructive: %s.\n", d.Summary())
		return
	}
	fmt.Fprintf(w, "This replace is DESTRUCTIVE: %s.\n", d.Summary())
}

// confirmDestructiveReplace implements rungs 2 through 5 of the precedence
// ladder. It returns nil when the PUT may proceed and an error — a non-zero
// exit — when it may not.
//
// WHY THIS IS NEW CODE AND NOT A CALL INTO THE SHARED HELPER.
//
// internal/resource.ConfirmDelete returns proceed when the skip flag is set,
// when JSON mode is set, AND when stdin is not a TTY. That last clause is the
// exact inverse of what D-27-05 requires here: a non-interactive run is
// precisely where a destructive replace must refuse, because there is nobody
// to notice. Worse, under `go test` stdin is never a TTY — so a guard built on
// that helper would clobber every contract in CI while passing a test that
// asserted only "an error mentions --yes". That is structurally the same
// failure Phase 26 found five times: a control sitting green under the very
// mutation it exists to catch.
//
// The overwrite confirmation in cmd/invoices (confirmOverwrite) carries the
// same inverted gating and is not a valid substitute either. Its prompt
// MECHANICS are worth copying and are copied below — it writes to the
// command's error writer rather than to the process stderr, which makes the
// prompt testable, and it returns an error on abort rather than nil, which is
// the non-zero exit wanted here (cmd/teams/delete.go returns nil on abort;
// that is the wrong shape for this command).
//
// Without this comment a future reader will simplify these four rungs back
// into the one-line shared call and reintroduce the defect silently. That is
// the single highest-value mutation in this phase, and it is red in
// TestEconomicsSetRefusesDestructive.
func confirmDestructiveReplace(c *cobra.Command, fileArg string, d economicsDiff) error {
	// RUNG 2 — nothing declared in the current contract would be lost.
	// Proceed with no confirmation and no noise: an additive edit is the
	// common case and must not train the operator to reflex-type --yes.
	if !d.IsDestructive() {
		return nil
	}

	// RUNG 3 — the operator consented explicitly. Render the loss anyway, for
	// the record: --yes means "I accept this", not "do not tell me".
	if cmd.YesMode() {
		renderDestructiveDiff(c.ErrOrStderr(), d)
		return nil
	}

	// RUNG 4 — destructive, no --yes, and nobody is watching. REFUSE. Never
	// prompt.
	//
	// The `-` case is tested EXPLICITLY rather than left to the TTY check, for
	// two independent reasons. First, stdin was already consumed by the
	// document, so there is nothing left to read and a prompt would either
	// block or read EOF. Second, D-27-05's planning note requires the
	// behaviour be explicit rather than emergent: `--file -` in a terminal
	// still has a TTY on fd 0 in some shells, and inferring the refusal from
	// the TTY probe alone would make it depend on the shell.
	//
	// Prompting into a pipe is not a softer alternative. A warning nobody must
	// acknowledge is scrolled past, and by the time it is read the loss is
	// already committed.
	if fileArg == "-" || !term.IsTerminal(os.Stdin.Fd()) {
		renderDestructiveDiff(c.ErrOrStderr(), d)
		return fmt.Errorf("refusing to replace the economics contract: %s.\n"+
			"Re-run with --yes to proceed, or with --dry-run to preview the replacement.", d.Summary())
	}

	// RUNG 5 — destructive, no --yes, stdin is a TTY. Ask.
	renderDestructiveDiff(c.ErrOrStderr(), d)
	fmt.Fprint(c.ErrOrStderr(), "Replace the economics contract anyway? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("aborted: %s", d.Summary())
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted: %s", d.Summary())
	}
	return nil
}
