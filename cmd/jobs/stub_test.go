package jobs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/revenium/revenium-cli/internal/api"
)

// newStubServer starts a hermetic httptest server for handler and registers
// its shutdown with t.
//
// D-27-12's testing posture: response bodies handed to these stubs are
// hand-built to match the dev spec's schemas. They are never a decode of the
// cached OpenAPI document and never a frozen schema fixture under testdata —
// a fixture that drifts silently is worse than no fixture, because it proves
// the CLI still agrees with a snapshot rather than with the contract.
func newStubServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// stubClient builds an api.Client pointed at a stub server.
//
// The third positional argument of api.NewClient is TeamID. Every cmd/jobs
// test written before Phase 27 passes "" there, which the requireTeam() guard
// now refuses — so any new happy-path test must pass a real team or it will
// fail against the guard rather than against the behaviour it means to test.
func stubClient(url, teamID string) *api.Client {
	return api.NewClient(url, "test-key", teamID, "", "", false)
}

// requestCounter counts the requests a stub handler observed. Increments are
// mutex-guarded: httptest serves each request on its own goroutine, so an
// unguarded counter would trip the race detector and report a data race
// instead of the guard regression the test exists to diagnose.
type requestCounter struct {
	mu sync.Mutex
	n  int
}

func (c *requestCounter) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *requestCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// execJobs runs args through the package-level Cmd — the same object main.go
// registers — so the command under test is reached through the real
// registration path rather than through a directly constructed command.
//
// Executing a child command object directly is not an option: cobra's
// Command.ExecuteC redirects to Root() whenever the receiver has a parent
// (cobra v1.10.2, command.go:1089-1092), so args set on a registered child are
// ignored and the root's args are used instead. Setting them on Cmd is both
// the correct call and the stronger assertion.
func execJobs(out io.Writer, args ...string) error {
	Cmd.SetOut(out)
	Cmd.SetErr(out)
	Cmd.SetArgs(args)
	return Cmd.Execute()
}

// economicsResourceBody is a hand-built JobTypeEconomicsResource carrying all
// seven JobTypeEconomicsRequest properties (including overheadCurrency,
// correction C1) plus the two read-only keys jobType and currentBaseline.
//
// overheadPerUnit is deliberately 1234567: below 1e6 the scientific-notation
// bug that money() exists to prevent does not appear, so a fixture of small
// round numbers would leave the rendering path unpinned.
const economicsResourceBody = `{
  "jobType": "acme-review",
  "unitMetricKey": "documents-reviewed",
  "unitLabel": "documents reviewed",
  "metrics": [
    {"key":"documents-reviewed","type":"COUNT","direction":"HIGHER_IS_BETTER","aggregation":"SUM","resolution":"PER_JOB"}
  ],
  "dimensions": [
    {"key":"region","allowedValues":["us-east","eu-west"]}
  ],
  "monetization": {
    "metricKey":"documents-reviewed",
    "valuePerUnit":12.5,
    "currency":"USD",
    "category":"COST_AVOIDED",
    "basis":"REALIZED"
  },
  "overheadPerUnit": 1234567,
  "overheadCurrency": "USD",
  "currentBaseline": {
    "version": 3,
    "costPerUnit": 4.25,
    "minutesPerUnit": 18,
    "qualityRate": 0.955,
    "hourlyRate": 85.5,
    "currency": "USD",
    "provenance": "CUSTOMER_DECLARED",
    "declaredBy": "ops@acme.example",
    "evidenceUrl": "https://acme.example/evidence/3",
    "effectiveFrom": "2026-01-01T00:00:00Z"
  }
}`
