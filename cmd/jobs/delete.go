package jobs

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	rerrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/revenium/revenium-cli/internal/resource"
)

// init wires the delete subcommand onto the package-level Cmd parent.
// Multiple init() per package is core Go — Plan 01 wires list+get in jobs.go,
// each subsequent plan adds its own AddCommand call in its own file (D-21).
func init() {
	Cmd.AddCommand(newDeleteCmd())
}

// newDeleteCmd builds the `revenium jobs delete <agenticJobId>...` subcommand.
//
// The command's ARITY picks the endpoint (JOBS-17, D-28-03):
//
//	one id   -> DELETE /v2/api/jobs/<url.PathEscape(id)>   (single-item, D-22+D-25)
//	two+ ids -> DELETE /v2/api/jobs  with {"ids": [...]}    (bulk_delete_jobs)
//
// Exactly one id keeps today's behaviour byte-for-byte — the same path, the
// same dry-run label, the same "Deleted job <id>." line, and NO team guard —
// because the shipped invocation is documented and scripted against. The bulk
// arm is new surface and carries its own guard: that endpoint declares teamId
// required, and the client appends the parameter only when a team resolved, so
// without the guard a mass delete goes out team-less and the operator reads the
// resulting 4xx as "my ids were wrong" (D-28-12).
//
// The single-id branch previously relied on an exactly-one-argument validator
// to make the bulk path unreachable; the validator is now a one-or-more form,
// so that guarantee has moved into the dispatcher's arity test. delete_test.go
// pins it from the outside by asserting the URL path per arity — the single
// defect this shape exists to prevent is one path variable serving both
// endpoints, which no "the delete succeeded" assertion can see.
//
// WHY TWO FUNCTIONS AND NOT ONE BRANCHING BODY: tools/coverage-audit builds one
// identifier scope per FUNCTION body, descending into nested blocks, and keeps
// only the first binding of a given name. Two `path :=` in one RunE would both
// resolve to the first, and the bulk endpoint would be recorded under the
// single-item URL with nothing erroring (RESEARCH Pitfall 3).
func newDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <agenticJobId>...",
		Short:       "Delete one or more jobs",
		Annotations: map[string]string{"mutating": "true"},
		// ValidResourceID already loops over every positional argument, so
		// widening the count validator needs no change to the id validator.
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a job (with confirmation)
  revenium jobs delete loan-app-12345

  # Delete without confirmation
  revenium jobs delete loan-app-12345 --yes

  # Delete several jobs in one request
  revenium jobs delete loan-app-1 loan-app-2 loan-app-3

  # Preview a bulk delete without calling the API
  revenium jobs delete loan-app-1 loan-app-2 --dry-run`,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 1 {
				return deleteOneJob(c, args[0])
			}
			return deleteJobsBulk(c, args)
		},
	}

	return c
}

// deleteOneJob is the shipped single-id behaviour, moved verbatim out of RunE
// and otherwise untouched: DELETE /v2/api/jobs/{id}, the same dry-run label,
// the same confirmation subject, the same success line.
//
// It deliberately carries NO team guard. D-28-03's byte-for-byte promise is the
// narrower and more explicit constraint, and the three shipped tests for this
// path pass an empty team id; the single-id gap belongs to the recorded
// deferred item "a repo-wide missing-team guard" (T-27-18), not here.
func deleteOneJob(c *cobra.Command, id string) error {
	// Capture path ONCE so dry-run preview and real DELETE are byte-identical.
	path := fmt.Sprintf("/v2/api/jobs/%s", url.PathEscape(id))

	if cmd.DryRun() {
		return dryrun.Render(cmd.Output, "delete", "job", path, nil)
	}

	yes, _ := c.Flags().GetBool("yes")

	ok, err := resource.ConfirmDelete("job", id, yes, cmd.Output.IsJSON())
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
		return err
	}

	if !cmd.Output.IsQuiet() {
		fmt.Fprintf(c.OutOrStdout(), "Deleted job %s.\n", id)
	}
	return nil
}

// deleteJobsBulk is a SEPARATE FUNCTION, not an else-branch, so its `path`
// local lives in its own function scope — see the note on newDeleteCmd for what
// tools/coverage-audit does to two bindings of one name in a single body. The
// split additionally keeps `path` a local at the request call site, which is the
// constraint cmd/jobs/economics_set.go already records: a path arriving as a
// PARAMETER resolves to nothing and fails the extractor's tree test. Never pass
// the path into a shared helper.
func deleteJobsBulk(c *cobra.Command, ids []string) error {
	// The bulk endpoint declares teamId required and the client appends it only
	// when a team resolved, so this refusal is what stops a mass deletion from
	// going out unscoped (D-28-12).
	if err := requireTeam(); err != nil {
		return err
	}

	path := "/v2/api/jobs"
	// The ids travel in the order the operator typed them, with no
	// deduplication and no sorting: what was asked for is what is sent.
	body := map[string]interface{}{"ids": ids}

	if cmd.DryRun() {
		// dryrun.Render's table path formats the body with a general-purpose
		// verb, and a Go map rendering is not a preview anyone reads — so the
		// human path gets the joined id list and only --json gets the document
		// that will actually be sent (RESEARCH Pitfall 8).
		if cmd.Output.IsJSON() {
			return dryrun.Render(cmd.Output, "delete", "jobs", path, body)
		}
		return dryrun.Render(cmd.Output, "delete", "jobs", path, strings.Join(ids, ", "))
	}

	// --yes is a ROOT persistent flag (cmd/root.go) read through this accessor,
	// which returns the very variable the flag is bound to. The single-id arm
	// above reads it off the command instead; that form is preserved there only
	// because its behaviour is frozen byte-for-byte, and it is not the form to
	// copy — jobs.Cmd is exercised without its root parent, so a flag read off
	// the command is invisible to a test that sets the root flag.
	yes := cmd.YesMode()

	// The shared helper, reused UNCHANGED. It auto-confirms under --yes, in
	// JSON mode, and off a terminal. That third rung is correct here and is NOT
	// a contradiction of the fail-closed refusal Phase 27 built for the
	// economics replace: that guarded a silent clobber of data the operator
	// never named, whereas here every id is typed on the command line. Do not
	// "reconcile" the two by hardening this one.
	ok, err := resource.ConfirmDelete("jobs", deleteSubject(ids), yes, cmd.Output.IsJSON())
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// The result argument is NON-NIL here, and correctly so: this endpoint
	// answers 200 with a BulkDeleteResponse_Read body, and the deletedIds in
	// that body are what the whole fail-closed report below is computed from.
	// The two append endpoints in this phase answer 201 with no body at all and
	// must pass nil — Do decodes whenever the result argument is non-nil, so the
	// two call shapes are opposites and must not be unified.
	//
	// What a DECODE FAILURE means here is the other half of that story, and it
	// is not the same as it is for the append commands. A server that answers
	// 2xx with nothing decodable — a 204, an empty 200 — may well have DELETED
	// EVERY JOB. Reporting that as a transport failure tells a script that
	// nothing happened, on an irreversible operation where the opposite may be
	// true; so a decode failure is treated as "nothing could be confirmed" and
	// routed into the shortfall report, never returned as a malformed-response
	// error and never as success (WR-02).
	var resp map[string]interface{}
	if err := cmd.APIClient.Do(c.Context(), "DELETE", path, body, &resp); err != nil {
		// Scoped to the DECODE STAGE ONLY, and discriminated by TYPE FIRST.
		//
		// internal/api/client.go returns mapHTTPError — always a
		// *rerrors.APIError — for any status at or above 400, BEFORE it decodes
		// anything. So an error of that type provably did not come from the
		// decode stage, however its text happens to read, and excluding the
		// type outright is what makes the second test safe to apply at all.
		//
		// That ordering is not a nicety. mapHTTPError's default arm (every 4xx
		// except 401/403/404) builds its message by copying the server's own
		// `message` field verbatim, so the error text of a 400 is REMOTE-
		// CONTROLLED: a body of {"message":"upstream failed to decode response
		// from job service"} satisfies the string test below on text this
		// process never wrote. Matching text alone therefore reported a request
		// the API refused as an unconfirmed delete — discarding the actionable
		// message and degrading the exit code from ExitValidation to
		// ExitGeneral (CR-01).
		//
		// What survives the type test is a sub-400 response whose body would
		// not decode, and the wrapper string is the only discriminator
		// internal/api exposes for that stage. It is matched against an error
		// THIS PROCESS constructed (client.go:174-178), never against server
		// text. A 4xx, a 5xx, a DNS failure and a refused connection all fall
		// through and are returned unchanged: transport failures reach the
		// operator through networkError's "Could not connect to ..." text,
		// which cannot satisfy either test. Widening the tolerance until a real
		// error reads as a clean delete would be a worse defect than the one
		// being fixed (T-28G-05).
		var apiErr *rerrors.APIError
		if errors.As(err, &apiErr) || !strings.Contains(err.Error(), "failed to decode response") {
			return err
		}

		// Discard anything a partial decode may have left behind — a body that
		// would not decode has confirmed nothing — and let the normalisation
		// below turn the nil map into the empty one and warn the operator.
		resp = nil
	}

	// Reached on BOTH routes, which is the whole reason the normalisation lives
	// here rather than inside the branch above. A 2xx body of the literal `null`
	// DECODES CLEANLY into a nil map: err is nil, the tolerance never runs, and
	// a substitution written inside it never fires for the one 2xx shape that
	// reaches this point nil by a route other than a decode failure (WR-03).
	//
	// EMPTY but NON-NIL: reportBulkDelete renders before it decides, and a nil
	// map encodes to the literal `null` under --json, which is not a document a
	// consumer can index. An empty map renders the same empty row the table path
	// already handles and emits `{}`.
	//
	// The warning is deliberately the SAME on both routes, because the
	// operator's situation is the same: the server answered, and confirmed
	// nothing. Splitting it into a `null`-specific message would give two names
	// to one condition.
	if resp == nil {
		resp = map[string]interface{}{}
		fmt.Fprintln(c.ErrOrStderr(),
			"warning: the server answered with no readable body, so no deletion could be confirmed; "+
				"the jobs may or may not have been deleted.")
	}

	// Falls through unchanged on both paths. With no deletedIds to read,
	// reportBulkDelete marks every requested id missing and returns the
	// non-zero "delete incomplete:" error — the fail-closed posture D-28-10
	// asks for. There is deliberately no success path and no early return here.
	return reportBulkDelete(c, ids, resp)
}

// maxPromptIDs is where `Delete jobs a, b, c, ...? [y/N]` stops being readable.
// It is a DISPLAY rule only: every id is still sent, and the tail count says how
// many are hidden.
const maxPromptIDs = 10

// deleteSubject renders the confirmation subject for a bulk delete.
//
// The remainder phrase is plain ASCII rather than an ellipsis character on
// purpose: the prompt is written to stderr with no terminal-capability probe,
// and a non-UTF-8 terminal would render "…" as mojibake in the one line that
// must be unambiguous before a destructive act (D-28-09).
func deleteSubject(ids []string) string {
	if len(ids) <= maxPromptIDs {
		return strings.Join(ids, ", ")
	}
	return fmt.Sprintf("%s and %d more",
		strings.Join(ids[:maxPromptIDs], ", "), len(ids)-maxPromptIDs)
}

// bulkDeleteTableDef is the single-row layout for a bulk delete result.
//
// StatusColumn: -1 disables status colorization — no cell here carries status
// semantics. Every key of BulkDeleteResponse_Read remains reachable under
// --json, which receives the untouched server payload.
var bulkDeleteTableDef = output.TableDef{
	Headers:      []string{"Requested", "Deleted", "Deleted IDs", "Message"},
	StatusColumn: -1,
}

// toBulkDeleteRows builds the single result row.
//
// Cell order is an explicit ordered literal rather than a range over the
// response map, which would produce non-deterministic Go map iteration order.
// Both count cells go through num(), never str(): str() uses the unformatted
// fmt print family, which emits 1.234567e+06 for any JSON-decoded number at or
// above 1e6.
func toBulkDeleteRows(resp map[string]interface{}) [][]string {
	return [][]string{{
		num(resp, "requestedCount", "%.0f"),
		num(resp, "deletedCount", "%.0f"),
		strings.Join(deletedIDs(resp), ", "),
		str(resp, "message"),
	}}
}

// deletedIDs reads the server's confirmed-deleted list, keeping only the
// elements that are actually strings.
//
// A missing, null or non-array field yields an EMPTY list rather than an
// error, and the caller treats an empty list as "nothing is confirmed" — an
// absent field must never read as success.
func deletedIDs(resp map[string]interface{}) []string {
	arr, ok := resp["deletedIds"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// reportBulkDelete renders the server's answer and then decides whether the
// operation actually succeeded.
//
// ORDER IS LOAD-BEARING: the render happens FIRST and the error is returned
// AFTER. main.go renders a returned error to stderr and exits, so returning
// early on a shortfall would emit the error and no payload — and D-28-10
// requires that under --json the server's own document still reaches stdout
// while the command exits non-zero (RESEARCH Pitfall 7).
func reportBulkDelete(c *cobra.Command, requested []string, resp map[string]interface{}) error {
	if err := cmd.Output.Render(bulkDeleteTableDef, toBulkDeleteRows(resp), resp); err != nil {
		return err
	}

	// Set MEMBERSHIP, not a positional or length comparison. The server may
	// return its ids in any order, and an operator who repeated an id has still
	// had that id deleted — so a length comparison would report a false
	// shortfall on a duplicated list. The server's own requestedCount and
	// deletedCount are displayed as it reported them but are never the trigger,
	// for that same reason.
	confirmed := make(map[string]bool, len(requested))
	for _, id := range deletedIDs(resp) {
		confirmed[id] = true
	}

	// Walk the ids the operator named, in order, skipping repeats: an id typed
	// twice is one job, and it must not be reported twice either.
	var named, missing []string
	seen := make(map[string]bool, len(requested))
	for _, id := range requested {
		if seen[id] {
			continue
		}
		seen[id] = true
		named = append(named, id)
		if !confirmed[id] {
			missing = append(missing, id)
		}
	}

	// The MIRROR of deletedIDs's doc comment. That one records that an absent
	// field must never read as success; this records that a SURPLUS field must
	// never read as plain success either. The shipped check computed only
	// requested − confirmed, so an id the operator never named — a server-side
	// prefix match, an id collision, a wrong-team answer — was discarded and the
	// command exited 0. On an irreversible operation an unrequested deletion is
	// at least as serious as an unconfirmed one (WR-03).
	//
	// Order is the server's, so the message is reproducible, and de-duplicated
	// so a server that repeats an unrequested id reports it once. Computed
	// against `seen` — the ids the operator NAMED — so a collapsed duplicate
	// (a a b -> a b) is not mistaken for a surplus.
	var surplus []string
	reported := make(map[string]bool)
	for _, id := range deletedIDs(resp) {
		if seen[id] || reported[id] {
			continue
		}
		reported[id] = true
		surplus = append(surplus, id)
	}
	// Written UNCONDITIONALLY, deliberately not gated on IsQuiet(): --quiet
	// exists to suppress the routine "Deleted N jobs." line, and an unrequested
	// destructive deletion is the opposite of routine. Gating it would let a
	// scripted caller silence the only signal that it happened (T-28G-09).
	//
	// It is also NOT folded into the returned error and NOT fatal: it takes a
	// misbehaving server to produce one, and a non-zero exit would break every
	// operator whose server legitimately returns extra ids. The exit code below
	// stays governed solely by `missing`. This write sits after the render and
	// before that decision so it reaches both the success and shortfall paths.
	if len(surplus) > 0 {
		fmt.Fprintf(c.ErrOrStderr(),
			"warning: the server reports deleting %d job%s this command never requested: %s\n",
			len(surplus), pluralSuffix(len(surplus)), strings.Join(surplus, ", "))
	}

	if len(missing) == 0 {
		if !cmd.Output.IsQuiet() {
			fmt.Fprintf(c.OutOrStdout(), "Deleted %d job%s.\n",
				len(named), pluralSuffix(len(named)))
		}
		return nil
	}

	// A partial delete is a FAILURE. The plain error maps to the general
	// failure exit code, which is the right one here, and a script reading the
	// exit status learns that not every job it named is gone.
	detail := ""
	if m, ok := resp["message"].(string); ok && strings.TrimSpace(m) != "" {
		detail = ": " + m
	}
	return fmt.Errorf("delete incomplete: %d of %d job%s not deleted (%s)%s",
		len(missing), len(named), pluralSuffix(len(named)),
		strings.Join(missing, ", "), detail)
}
