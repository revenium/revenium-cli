package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/x/term"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	rerrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteJobWithYes — standard happy path with --yes.
//
// Asserts:
//   - HTTP method is exactly "DELETE" (literal, not method-helper-erased)
//   - URL path is /v2/api/jobs/loan-app-1 (the single-item form, D-22+D-25)
//   - URL path is NOT /v2/api/jobs (RESEARCH §Risk 4: bulk-delete collision guard)
//   - Output contains "Deleted job loan-app-1." exactly (pattern S8 / D-12)
func TestDeleteJobWithYes(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		// Defensive: never hit the bulk-delete endpoint (RESEARCH §Risk 4).
		// ExactArgs(1) prevents zero-arg invocation at the cobra layer, but this
		// runtime assertion catches future regressions where someone might
		// accidentally drop the path-suffix interpolation.
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDeleteCmd()
	// Pattern S10: re-register --yes as a local flag for the standalone
	// subcommand test. At runtime the flag is inherited from rootCmd as a
	// persistent flag — but tests instantiate the subcommand in isolation.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.Contains(t, buf.String(), "Deleted job loan-app-1.")
}

// TestDeleteJobQuiet — confirms --quiet suppresses the "Deleted job ..." line
// (pattern S8). The DELETE call still fires; only the stdout summary is muted.
func TestDeleteJobQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 4th arg true = quiet=true.
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// TestDeleteJobJSONMode — confirms JSON mode auto-confirms without --yes
// (D-12). The handler still sees DELETE on the single-item path.
func TestDeleteJobJSONMode(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v2/api/jobs/loan-app-1", r.URL.Path)
		assert.NotEqual(t, "/v2/api/jobs", r.URL.Path)
		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Deleted", "id": "loan-app-1"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	// 3rd arg true = jsonMode=true; ConfirmDelete should auto-confirm.
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newDeleteCmd()
	// Register --yes so the flag is parseable; the test does NOT pass it.
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.True(t, deleteCalled, "delete should proceed without prompt in JSON mode")
}

// ---------------------------------------------------------------------------
// Phase 28 / JOBS-17 — the arity split, its three confirmation rungs, and its
// fail-closed partial-delete report.
//
// Everything below this line is ADDED. The three tests above are the shipped
// single-id contract and are left byte-for-byte alone on purpose: they pass an
// EMPTY team id, and their continuing to pass unmodified is the proof that the
// team guard landed on the bulk arm ONLY.
// ---------------------------------------------------------------------------

// withTTYStdin points os.Stdin at a real terminal preloaded with answer, so
// ConfirmDelete takes its interactive rung and reads exactly that answer.
func withTTYStdin(t *testing.T, answer string) {
	t.Helper()
	master, slave := openPTY(t)
	require.True(t, term.IsTerminal(slave.Fd()),
		"the allocated pseudo-terminal must report as a terminal, or this test proves nothing")

	_, err := master.WriteString(answer + "\n")
	require.NoError(t, err, "priming the terminal with the operator's answer")

	prev := os.Stdin
	os.Stdin = slave
	t.Cleanup(func() { os.Stdin = prev })
}

// withCapturedProcessStderr redirects the PROCESS's stderr — where
// ConfirmDelete writes its prompt, which is not the command's error writer —
// into a pipe, and returns a reader for what landed there.
func withCapturedProcessStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	prev := os.Stderr
	os.Stderr = w

	drained := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		drained <- b.String()
	}()

	var once sync.Once
	var captured string
	read := func() string {
		once.Do(func() {
			os.Stderr = prev
			_ = w.Close()
			captured = <-drained
			_ = r.Close()
		})
		return captured
	}
	t.Cleanup(func() { read() })
	return read
}

// bulkDeleteStub records what the DELETE request carried and answers with a
// caller-supplied body.
//
// status is ADDITIVE and zero-valued-means-today's-behaviour: a stub that never
// sets it answers exactly as it did before the field existed, which is what lets
// the ten pre-existing bulk-delete and single-id tests stay unmodified. A
// non-zero status writes that status and NO body at all, which is the only way
// to serve the bodyless-2xx shapes (204, and 500 as the control) that no
// shipped stub could produce, because the body-only form always writes a body —
// even an empty one — with a 200.
type bulkDeleteStub struct {
	body   string
	status int

	hits requestCounter
	mu   sync.Mutex
	path string
	team string
	raw  []byte
}

func (s *bulkDeleteStub) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.hits.inc()
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "DELETE", r.Method)

		s.mu.Lock()
		s.path, s.team, s.raw = r.URL.Path, r.URL.Query().Get("teamId"), raw
		s.mu.Unlock()

		if s.status != 0 {
			w.WriteHeader(s.status)
			// A status-carrying stub may ALSO carry a body. mapHTTPError's
			// default arm echoes the server's own `message` field into the
			// error text, and the only way to prove that echoed text cannot
			// reach the decode tolerance is to let a 4xx stub supply one.
			// Writing the body only when it is non-empty leaves every
			// bodyless status stub (204, 500) byte-for-byte as it was.
			if s.body != "" {
				fmt.Fprint(w, s.body)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, s.body)
	}
}

func (s *bulkDeleteStub) recorded() (path, team string, raw []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path, s.team, append([]byte(nil), s.raw...)
}

// sentIDs decodes the "ids" array the CLI actually put on the wire.
func (s *bulkDeleteStub) sentIDs(t *testing.T) []string {
	t.Helper()
	_, _, raw := s.recorded()
	var doc struct {
		IDs []string `json:"ids"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc), "request body was: %s", raw)
	return doc.IDs
}

// setupBulkDelete wires the stub, the client and the formatter, and returns the
// stub plus the buffer the formatter writes to.
func setupBulkDelete(t *testing.T, respondWith string) (*bulkDeleteStub, *bytes.Buffer) {
	t.Helper()
	return setupBulkDeleteStub(t, &bulkDeleteStub{body: respondWith})
}

// setupBulkDeleteStub is setupBulkDelete for a stub the caller built itself —
// the form the status-carrying cases need. setupBulkDelete delegates to it, so
// there is exactly one place that wires the server, the client and the
// formatter and the two forms cannot drift apart.
func setupBulkDeleteStub(t *testing.T, stub *bulkDeleteStub) (*bulkDeleteStub, *bytes.Buffer) {
	t.Helper()
	srv := newStubServer(t, stub.handler(t))

	buf := &bytes.Buffer{}
	cmd.APIClient = stubClient(srv.URL, "team-28")
	cmd.Output = output.NewWithWriter(buf, buf, false, false)
	return stub, buf
}

// bulkDeleteOK is the shape BulkDeleteResponse_Read takes when everything the
// operator named was in fact deleted.
const bulkDeleteOK = `{"requestedCount":2,"deletedCount":2,"deletedIds":["a","b"],"message":"Deleted"}`

// TestDeleteRoutesByArity is THE test of this plan: the arity picks the
// endpoint, and the proof is the URL path per case.
//
// Mutation contract (28-VALIDATION.md): collapsing deleteOneJob and
// deleteJobsBulk into one RunE with a single shared `path` variable must turn
// the two-id case red HERE, on the path equality. A require.NoError alone stays
// green under a router that sends every arity to one endpoint — which is the
// exact green-for-nothing shape Phase 26 found five times and Phase 27 three
// more.
func TestDeleteRoutesByArity(t *testing.T) {
	cases := []struct {
		name     string
		ids      []string
		wantPath string
		wantIDs  []string // nil means the request must carry NO body at all
	}{
		{
			name:     "one id uses the single-item endpoint",
			ids:      []string{"a"},
			wantPath: "/v2/api/jobs/a",
		},
		{
			name:     "two ids use the bulk endpoint",
			ids:      []string{"a", "b"},
			wantPath: "/v2/api/jobs",
			wantIDs:  []string{"a", "b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withNonTTYStdin(t)
			stub, buf := setupBulkDelete(t, bulkDeleteOK)

			err := execJobs(buf, append([]string{"delete"}, tc.ids...)...)

			// The routing assertions come FIRST, and with assert rather than
			// require: a mutation that misroutes the request also tends to make
			// the command fail for a secondary reason, and a leading
			// require.NoError would abort on that instead of reporting the wrong
			// URL — which is the only diagnostic this test exists to deliver.
			//
			// Exactly one request per invocation: there is no per-id loop, so
			// an interrupted run cannot leave a partially-issued set.
			assert.Equal(t, 1, stub.hits.count(),
				"one invocation must issue exactly one DELETE, whatever the list length")

			gotPath, gotTeam, gotRaw := stub.recorded()
			assert.Equal(t, tc.wantPath, gotPath,
				"the ARITY picks the endpoint — a single shared path variable across both "+
					"arms is the defect this equality exists to catch")

			require.NoError(t, err, "output was: %s", buf.String())

			if tc.wantIDs == nil {
				assert.Empty(t, gotRaw,
					"the single-item endpoint takes no request body")
				return
			}
			assert.Equal(t, "team-28", gotTeam,
				"the bulk endpoint declares teamId required, so it must reach the query string")
			// ORDER IS LOAD-BEARING, and this is the second time it has been:
			// sentIDs calls require.NoError on json.Unmarshal, which ABORTS the
			// test on an empty body ("unexpected end of JSON input"). Below the
			// sentIDs call, this assertion could only ever execute in states
			// where it must pass — it was unfailable, and a mutation sending a
			// nil body went red on the ids equality one line up, which is a
			// different assertion with a different message (WR-04).
			assert.NotEmpty(t, gotRaw, "the bulk endpoint must carry a request body")
			assert.Equal(t, tc.wantIDs, stub.sentIDs(t),
				"the ids travel in the order the operator typed them")
		})
	}
}

// TestBulkDeleteConfirmPaths walks ConfirmDelete's three auto-confirm rungs, one
// case each, and asserts each still issues exactly one DELETE.
//
// The first two cases run on a REAL terminal preloaded with "n". That is what
// isolates the rung under test: on a terminal the helper would otherwise prompt
// and take that "n", so a case that deletes anyway proves the --yes rung (and
// then the JSON rung) fired on its own rather than falling through to the
// non-terminal shortcut the third case covers.
//
// Mutation contract: replacing ConfirmDelete with an unconditional true leaves
// ALL THREE of these green — every one of them is a confirm. Only
// TestBulkDeleteDeclined catches that mutation.
func TestBulkDeleteConfirmPaths(t *testing.T) {
	cases := []struct {
		name     string
		jsonMode bool
		setup    func(t *testing.T)
	}{
		{
			name: "--yes overrides a terminal that would decline",
			setup: func(t *testing.T) {
				withTTYStdin(t, "n")
				setRootFlag(t, "yes", "true")
			},
		},
		{
			name:     "JSON mode auto-confirms without --yes on a terminal that would decline",
			jsonMode: true,
			setup: func(t *testing.T) {
				withTTYStdin(t, "n")
			},
		},
		{
			name: "a non-terminal stdin auto-confirms with neither flag",
			setup: func(t *testing.T) {
				withNonTTYStdin(t)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			readStderr := withCapturedProcessStderr(t)
			tc.setup(t)

			stub, buf := setupBulkDelete(t, bulkDeleteOK)
			cmd.Output = output.NewWithWriter(buf, buf, tc.jsonMode, false)

			err := execJobs(buf, "delete", "a", "b")

			// Counter first, with assert — see the note in TestBulkDeleteDryRun.
			assert.Equal(t, 1, stub.hits.count(),
				"this confirmation rung must let the delete through")
			require.NoError(t, err, "output was: %s", buf.String())
			_ = readStderr()
		})
	}
}

// TestBulkDeleteDeclined is the ONLY test in this file that can catch a
// confirmation helper replaced by an unconditional true.
//
// It answers the real prompt with "n" over a real terminal. The load-bearing
// assertion is the request counter, not the returned error: declining is not a
// failure, so the command returns nil either way, and only the counter
// distinguishes "asked and obeyed" from "never asked".
func TestBulkDeleteDeclined(t *testing.T) {
	readStderr := withCapturedProcessStderr(t)
	withTTYStdin(t, "n")

	stub, buf := setupBulkDelete(t, bulkDeleteOK)

	err := execJobs(buf, "delete", "a", "b")

	// Counter first, with assert — see the note in TestBulkDeleteDryRun.
	assert.Equal(t, 0, stub.hits.count(),
		"a declined confirmation must issue NO delete request at all")
	require.NoError(t, err, "declining is not an error; output was: %s", buf.String())

	// The subject the operator was asked about names both ids, on stderr.
	assert.Contains(t, readStderr(), "Delete jobs a, b?")
}

// TestBulkDeleteDryRun pins the other half of the mutating annotation: the
// command declares itself mutating AND actually honours --dry-run.
//
// Mutation contract: deleting the `if cmd.DryRun()` early return must turn this
// red on the COUNTER. Neither the exit status nor the printed text can catch it
// — the dry-run renderer produces its output either way.
func TestBulkDeleteDryRun(t *testing.T) {
	withNonTTYStdin(t)
	setRootFlag(t, "dry-run", "true")

	stub, buf := setupBulkDelete(t, bulkDeleteOK)

	err := execJobs(buf, "delete", "loan-1", "loan-2")

	// The counter asserts FIRST, and with assert rather than require: under a
	// mutation that lets the request through, a leading require.NoError aborts
	// the test on whatever the server's answer happened to provoke, and the one
	// assertion this test exists for never runs (the shape 27-05 recorded as
	// deviation 4 and team_test.go carries the same note).
	assert.Equal(t, 0, stub.hits.count(),
		"--dry-run must issue no DELETE at all — this command is irreversible")
	require.NoError(t, err, "output was: %s", buf.String())

	out := buf.String()
	assert.Contains(t, out, "/v2/api/jobs", "the preview must name the bulk path")
	assert.Contains(t, out, "loan-1", "the preview must name every id")
	assert.Contains(t, out, "loan-2", "the preview must name every id")
}

// TestBulkDeletePartial pins the repudiation threat: a partial delete is NOT a
// success.
//
// Mutation contract: changing the shortfall branch to `return nil` must turn
// both modes red on the RETURNED ERROR. Asserting printed text would not — the
// server's payload and the row are rendered before the decision either way,
// which is the whole point of the render-first ordering.
func TestBulkDeletePartial(t *testing.T) {
	const partial = `{"requestedCount":3,"deletedCount":2,"deletedIds":["a","b"],"message":"c not found"}`

	t.Run("table mode names the id that did not go", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t, partial)

		err := execJobs(buf, "delete", "a", "b", "c")

		require.Error(t, err, "a partial delete must not succeed; output was: %s", buf.String())
		assert.Contains(t, err.Error(), "(c)", "the error must name the id that did not go")
		assert.Contains(t, err.Error(), "c not found", "the server's own message must survive")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("json mode emits the payload to stdout AND fails", func(t *testing.T) {
		stub := &bulkDeleteStub{body: partial}
		srv := newStubServer(t, stub.handler(t))

		// stdout is captured SEPARATELY from cobra's error writer: the property
		// under test is that the server's document reached stdout while the
		// command still failed, and one shared buffer cannot tell the two apart.
		var stdout, stderr, cobraOut bytes.Buffer
		cmd.APIClient = stubClient(srv.URL, "team-28")
		cmd.Output = output.NewWithWriter(&stdout, &stderr, true, false)

		err := execJobs(&cobraOut, "delete", "a", "b", "c")

		require.Error(t, err, "a partial delete must exit non-zero even under --json")
		assert.Contains(t, err.Error(), "(c)")

		// Unmarshal rather than substring-match: a substring assertion cannot
		// tell the server's document from the error text.
		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc),
			"stdout must carry the server's own JSON document; it was: %s", stdout.String())
		assert.Equal(t, []interface{}{"a", "b"}, doc["deletedIds"],
			"the payload must reach stdout verbatim, not be lost to the error path")
		assert.Equal(t, 1, stub.hits.count())
	})
}

// TestBulkDeletePartialUnconfirmable fails closed on an ABSENT deletedIds.
//
// A server that answers 200 with counts but no id list has confirmed nothing,
// and reading a missing field as success is precisely the shape that lets a
// script believe five deletions happened when none did.
func TestBulkDeletePartialUnconfirmable(t *testing.T) {
	withNonTTYStdin(t)
	stub, buf := setupBulkDelete(t, `{"requestedCount":3,"deletedCount":3}`)

	err := execJobs(buf, "delete", "a", "b", "c")

	require.Error(t, err, "an absent deletedIds must not read as success; output was: %s", buf.String())
	assert.Contains(t, err.Error(), "(a, b, c)",
		"every requested id is unconfirmed when the server names none")
	assert.Equal(t, 1, stub.hits.count())
}

// TestBulkDeleteDuplicateIDs — a repeated id is sent verbatim and a server that
// collapses it is NOT reported as a shortfall.
//
// This is what forces the shortfall check to be set membership rather than a
// length or positional comparison: requestedCount 3 against deletedCount 2 is a
// full success here, because every id the operator named was deleted.
func TestBulkDeleteDuplicateIDs(t *testing.T) {
	withNonTTYStdin(t)
	stub, buf := setupBulkDelete(t,
		`{"requestedCount":3,"deletedCount":2,"deletedIds":["a","b"],"message":"Deleted"}`)

	err := execJobs(buf, "delete", "a", "a", "b")
	require.NoError(t, err,
		"every id the operator named was deleted, so this is not a shortfall; output was: %s", buf.String())

	assert.Equal(t, []string{"a", "a", "b"}, stub.sentIDs(t),
		"the CLI neither deduplicates nor reorders the list it was given")
	assert.Equal(t, 1, stub.hits.count())
}

// TestBulkDeleteBodylessSuccess pins WR-02: a 2xx carrying no decodable body is
// an UNCONFIRMED delete, never a transport failure and never a success.
//
// The delete may well have happened — the server said 2xx — so the only honest
// report is the one reportBulkDelete already gives for an absent deletedIds:
// name every requested id as unconfirmed and exit non-zero. Reporting
// "failed to decode response: EOF" instead tells a script that nothing was
// deleted, on an irreversible operation where that may be flatly untrue.
//
// The 500 subtest is the CONTROL and is not optional: it is the only assertion
// that bounds the tolerance. A fix that swallowed every error from Do would
// leave the three bodyless subtests green while turning a server outage into a
// clean-looking shortfall report — a worse defect than WR-02 itself (T-28G-05).
func TestBulkDeleteBodylessSuccess(t *testing.T) {
	t.Run("an empty-body 200 confirms nothing and fails closed", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t, "")

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err,
			"a 2xx with no readable body has confirmed nothing; output was: %s", buf.String())
		assert.Contains(t, err.Error(), "delete incomplete",
			"the bodyless answer must route into the fail-closed shortfall report")
		assert.Contains(t, err.Error(), "(a, b)",
			"every requested id is unconfirmed when the server confirms none")
		assert.NotContains(t, err.Error(), "failed to decode response",
			"the operator must be told nothing could be confirmed, not that the response was malformed")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("a 204 with no body is identical to the empty 200", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDeleteStub(t, &bulkDeleteStub{status: http.StatusNoContent})

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err,
			"a 204 confirms no id at all; output was: %s", buf.String())
		assert.Contains(t, err.Error(), "delete incomplete")
		assert.Contains(t, err.Error(), "(a, b)")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("json mode still emits a JSON document to stdout AND fails", func(t *testing.T) {
		withNonTTYStdin(t)
		stub := &bulkDeleteStub{body: ""}
		srv := newStubServer(t, stub.handler(t))

		// stdout is captured SEPARATELY from cobra's error writer for the same
		// reason TestBulkDeletePartial's json subtest does it: the property
		// under test is what reached stdout, and one shared buffer cannot tell
		// the payload from the note written to c.ErrOrStderr().
		var stdout, stderr, cobraOut bytes.Buffer
		cmd.APIClient = stubClient(srv.URL, "team-28")
		cmd.Output = output.NewWithWriter(&stdout, &stderr, true, false)

		err := execJobs(&cobraOut, "delete", "a", "b")

		require.Error(t, err, "a bodyless 2xx must exit non-zero even under --json")
		assert.Contains(t, err.Error(), "(a, b)")

		// D-28-10's render-before-error ordering must survive the tolerance:
		// stdout carries a document even though the server sent none. The
		// literal `null` a nil map would encode to is NOT a document — a
		// consumer piping this into jq gets nothing it can index.
		assert.NotEqual(t, "null", strings.TrimSpace(stdout.String()),
			"a nil result map encodes to the literal null; the bodyless branch must substitute an empty non-nil map")
		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc),
			"stdout must carry a valid JSON document; it was: %s", stdout.String())
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("a 500 is returned unchanged and is not converted into a shortfall", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDeleteStub(t, &bulkDeleteStub{status: http.StatusInternalServerError})

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err, "a 500 is a real failure; output was: %s", buf.String())
		assert.NotContains(t, err.Error(), "delete incomplete",
			"THE control for T-28G-05: a genuine HTTP error must never be dressed up as a shortfall report")
		assert.Contains(t, err.Error(), "Revenium API error",
			"the mapped HTTP error text must survive to the operator unchanged")
		assert.Equal(t, 1, stub.hits.count())
	})

	// THE control the 500 subtest cannot be. 5xx takes mapHTTPError's
	// fixed-message arm and never echoes the server, so it can never carry the
	// decode wrapper's phrasing. Every 4xx except 401/403/404 takes the arm
	// that copies the server's own `message` field into the error text — which
	// makes that text REMOTE-CONTROLLED. A tolerance discriminated by string
	// match therefore fires on a request the API refused: the actionable
	// message is discarded, the operator is told the jobs "may or may not have
	// been deleted", and the exit code degrades from ExitValidation to
	// ExitGeneral. Only a TYPE test (*rerrors.APIError, which mapHTTPError
	// returns for every status at or above 400, before anything is decoded)
	// can bound it (CR-01).
	t.Run("a 400 whose own message quotes the decode wrapper is returned unchanged", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDeleteStub(t, &bulkDeleteStub{
			status: http.StatusBadRequest,
			body:   `{"message":"upstream failed to decode response from job service"}`,
		})

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err, "a 400 is a refusal, not a delete; output was: %s", buf.String())
		assert.NotContains(t, err.Error(), "delete incomplete",
			"server-supplied error text must not be able to trigger the decode tolerance")
		assert.NotContains(t, buf.String(), "may or may not have been deleted",
			"the API refused this request and deleted nothing; saying otherwise is a lie about an irreversible operation")
		assert.Contains(t, err.Error(), "HTTP 400",
			"the mapped HTTP error must survive to the operator unchanged")
		assert.Contains(t, err.Error(), "upstream failed to decode response from job service",
			"the server's own actionable message must not be thrown away")
		assert.Equal(t, rerrors.ExitValidation, rerrors.ExitCodeFor(err),
			"swallowing the *APIError degrades the exit code to ExitGeneral, so a script branching on it "+
				"reads a validation failure as a general one")
		assert.Equal(t, 1, stub.hits.count())
	})

	// The other bound: a TRANSPORT failure must not be tolerated either. A
	// server that accepts the connection and closes it without answering makes
	// http.Client return a *url.Error wrapping io.EOF. The tolerance must be
	// immune to that shape — and, more to the point, immune to it REMAINING
	// immune only by accident: an `errors.Is(err, io.EOF)` clause would start
	// matching here the day internal/api's networkError wraps its cause, which
	// is a routine improvement an error-wrapping linter actively asks for.
	t.Run("a transport-level EOF is returned unchanged and is not converted into a shortfall", func(t *testing.T) {
		withNonTTYStdin(t)

		var hits requestCounter
		srv := newStubServer(t, func(w http.ResponseWriter, r *http.Request) {
			hits.inc()
			conn, _, hijackErr := w.(http.Hijacker).Hijack()
			if !assert.NoError(t, hijackErr, "the stub must be able to drop the connection mid-request") {
				return
			}
			_ = conn.Close()
		})

		buf := &bytes.Buffer{}
		cmd.APIClient = stubClient(srv.URL, "team-28")
		cmd.Output = output.NewWithWriter(buf, buf, false, false)

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err, "a dropped connection is a real failure; output was: %s", buf.String())
		assert.NotContains(t, err.Error(), "delete incomplete",
			"a transport failure must never be dressed up as an unconfirmed delete")
		assert.NotContains(t, buf.String(), "may or may not have been deleted",
			"nothing reached the server, so nothing may have been deleted")
		assert.Contains(t, err.Error(), "Could not connect",
			"the transport error text must survive to the operator unchanged")
		assert.NotEqual(t, 0, hits.count(), "the request must actually have been attempted")
	})
}

// TestBulkDeleteNullBody pins WR-03: a 2xx body of the literal `null`.
//
// It is the one 2xx shape that reaches the renderer nil without ever touching
// the decode tolerance — `json.Decoder.Decode` on `null` SUCCEEDS and leaves the
// map nil — so an empty-map substitution written inside the error branch never
// runs for it. Under --json that nil map encodes to the literal `null`, which is
// not a document a consumer can index: exactly the outcome the substitution
// exists to prevent, arriving by the one route it did not cover.
func TestBulkDeleteNullBody(t *testing.T) {
	t.Run("a null body confirms nothing and fails closed", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t, "null")

		err := execJobs(buf, "delete", "a", "b")

		require.Error(t, err,
			"a body of `null` has confirmed nothing; output was: %s", buf.String())
		assert.Contains(t, err.Error(), "delete incomplete",
			"the null answer must route into the same fail-closed shortfall report as a bodyless 2xx")
		assert.Contains(t, err.Error(), "(a, b)",
			"every requested id is unconfirmed when the server confirms none")
		assert.Contains(t, buf.String(), "no deletion could be confirmed",
			"the operator must get the same warning as the bodyless case — the situation is the same")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("json mode emits an indexable document, never the literal null", func(t *testing.T) {
		withNonTTYStdin(t)
		stub := &bulkDeleteStub{body: "null"}
		srv := newStubServer(t, stub.handler(t))

		// stdout is captured SEPARATELY from cobra's error writer, as in the
		// other json subtests: the property under test is what reached stdout,
		// and one shared buffer cannot tell the payload from the warning.
		var stdout, stderr, cobraOut bytes.Buffer
		cmd.APIClient = stubClient(srv.URL, "team-28")
		cmd.Output = output.NewWithWriter(&stdout, &stderr, true, false)

		err := execJobs(&cobraOut, "delete", "a", "b")

		require.Error(t, err, "a null body must exit non-zero even under --json")
		assert.Contains(t, err.Error(), "(a, b)")

		assert.NotEqual(t, "null", strings.TrimSpace(stdout.String()),
			"a nil result map encodes to the literal null; jq gets nothing it can index")
		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc),
			"stdout must carry a valid JSON document; it was: %s", stdout.String())
		assert.Equal(t, 1, stub.hits.count())
	})
}

// TestBulkDeleteSurplus pins WR-03: the shortfall check runs in BOTH directions.
//
// requested − confirmed is a shortfall and is fatal. confirmed − requested is a
// SURPLUS — a job the operator never named that the server says it deleted, the
// signature of a server-side prefix match, an id collision or a wrong-team
// answer. On an irreversible operation that is at least as serious as a
// shortfall, and printing "Deleted 3 jobs." and exiting 0 makes it invisible.
//
// It is reported and NOT made fatal: it takes a misbehaving server to produce
// one, and a non-zero exit would break every operator whose server legitimately
// returns extra ids. The exit code stays governed solely by the shortfall.
func TestBulkDeleteSurplus(t *testing.T) {
	t.Run("a surplus id is named on stderr while the delete still succeeds", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t,
			`{"requestedCount":2,"deletedCount":3,"deletedIds":["a","b","z"],"message":"Deleted"}`)

		err := execJobs(buf, "delete", "a", "b")

		require.NoError(t, err,
			"every id the operator named was deleted, so this is not a failure; output was: %s", buf.String())
		out := buf.String()
		assert.Contains(t, out, "never requested: z",
			"an unrequested deletion must be named, not swallowed")
		assert.Contains(t, out, "Deleted 2 jobs.",
			"the operator's own two jobs are still reported as deleted")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("a clean delete produces no surplus line at all", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t, bulkDeleteOK)

		err := execJobs(buf, "delete", "a", "b")

		require.NoError(t, err, "output was: %s", buf.String())
		// THE assertion that stops the warning becoming unconditional noise: a
		// warning printed on every delete is a warning nobody reads.
		assert.NotContains(t, buf.String(), "never requested",
			"the surplus warning must appear only when there IS a surplus")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("a surplus and a shortfall are reported independently", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t,
			`{"requestedCount":3,"deletedCount":3,"deletedIds":["a","b","z"],"message":"c not found"}`)

		err := execJobs(buf, "delete", "a", "b", "c")

		require.Error(t, err,
			"c was named and not deleted, so this is still a shortfall; output was: %s", buf.String())
		assert.Contains(t, err.Error(), "(c)",
			"the exit code stays governed solely by the shortfall")
		assert.Contains(t, buf.String(), "never requested: z",
			"the surplus is reported on the shortfall path too, not only on the success path")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("a collapsed duplicate is not a surplus", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t,
			`{"requestedCount":3,"deletedCount":2,"deletedIds":["a","b"],"message":"Deleted"}`)

		err := execJobs(buf, "delete", "a", "a", "b")

		require.NoError(t, err, "output was: %s", buf.String())
		assert.NotContains(t, buf.String(), "never requested",
			"a repeated id is one job; the surplus set is computed against the ids the operator NAMED")
		assert.Equal(t, 1, stub.hits.count())
	})

	t.Run("--quiet suppresses the success line but never the surplus", func(t *testing.T) {
		withNonTTYStdin(t)
		stub, buf := setupBulkDelete(t,
			`{"requestedCount":2,"deletedCount":3,"deletedIds":["a","b","z"],"message":"Deleted"}`)
		// 4th arg true = quiet.
		cmd.Output = output.NewWithWriter(buf, buf, false, true)

		err := execJobs(buf, "delete", "a", "b")

		require.NoError(t, err, "output was: %s", buf.String())
		out := buf.String()
		assert.NotContains(t, out, "Deleted 2 jobs.",
			"--quiet suppresses the routine success line")
		assert.Contains(t, out, "never requested: z",
			"--quiet suppresses routine chatter, NOT the one signal that an unrequested destructive deletion occurred (T-28G-09)")
		assert.Equal(t, 1, stub.hits.count())
	})
}

// TestDeleteSubject pins the prompt subject: readable at any length, accurate
// about what it hides, and free of any byte a non-UTF-8 terminal would mangle.
func TestDeleteSubject(t *testing.T) {
	ids := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("job-%02d", i+1)
		}
		return out
	}

	cases := []struct {
		name          string
		ids           []string
		want          string
		wantRemainder string // "" means no remainder phrase may appear
	}{
		{name: "no ids", ids: nil, want: ""},
		{name: "one id", ids: []string{"job-01"}, want: "job-01"},
		{
			name: "exactly the threshold joins every id",
			ids:  ids(maxPromptIDs),
			want: strings.Join(ids(maxPromptIDs), ", "),
		},
		{
			name:          "past the threshold truncates and counts the remainder",
			ids:           ids(maxPromptIDs + 2),
			want:          strings.Join(ids(maxPromptIDs), ", ") + " and 2 more",
			wantRemainder: "and 2 more",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deleteSubject(tc.ids)
			assert.Equal(t, tc.want, got)

			if tc.wantRemainder == "" {
				assert.NotContains(t, got, " more",
					"a list at or under the threshold hides nothing, so it claims nothing")
			} else {
				assert.Contains(t, got, tc.wantRemainder,
					"the count of hidden ids must be accurate")
			}

			// Plain ASCII only. The prompt is written to stderr with no
			// terminal-capability probe, so a non-ASCII ellipsis could render
			// as mojibake in the one line that must be unambiguous before a
			// destructive act.
			for i := 0; i < len(got); i++ {
				require.Less(t, got[i], byte(0x80),
					"byte %d of %q is not ASCII", i, got)
			}
		})
	}
}
