package jobs

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOutcomeHistory is the happy-path test for
// `revenium jobs outcome-history <id>`. It pins the RES-07 contract: GET a
// bare JSON array from /v2/api/jobs/{id}/outcome/history, with NO page/size
// query params ever sent (the spec declares no pagination params on this
// operation).
func TestOutcomeHistory(t *testing.T) {
	var receivedPath string
	var receivedQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"sequence":1,"reportedAt":"2026-05-01T00:00:00Z","reportedBy":"ops@example.com","executionStatus":"SUCCESS","reason":"initial report"},
			{"sequence":2,"reportedAt":"2026-05-02T00:00:00Z","reportedBy":"ops@example.com","executionStatus":"SUCCESS","reason":"corrected value"}
		]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/jobs/loan-app-1/outcome/history", receivedPath)

	// No page/size query params present, regardless of table/JSON mode.
	_, hasPage := receivedQuery["page"]
	assert.False(t, hasPage, "outcome-history must NOT send a page query param")
	_, hasSize := receivedQuery["size"]
	assert.False(t, hasSize, "outcome-history must NOT send a size query param")

	assert.Contains(t, buf.String(), "initial report")
	assert.Contains(t, buf.String(), "corrected value")
}

// TestOutcomeHistoryNoPageFlags asserts the command exposes no
// --page/--page-size flags (i.e. AddListFlags was never called).
func TestOutcomeHistoryNoPageFlags(t *testing.T) {
	c := newOutcomeHistoryCmd()
	assert.Nil(t, c.Flags().Lookup("page"), "outcome-history must not register a --page flag")
	assert.Nil(t, c.Flags().Lookup("page-size"), "outcome-history must not register a --page-size flag")
}

// TestOutcomeHistoryJSON confirms JSON mode also sends no page/size params
// and round-trips raw revision data.
func TestOutcomeHistoryJSON(t *testing.T) {
	var receivedQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"sequence":1,"reportedAt":"2026-05-01T00:00:00Z","reportedBy":"ops@example.com","executionStatus":"SUCCESS","reason":"initial report"}]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newOutcomeHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	_, hasPage := receivedQuery["page"]
	assert.False(t, hasPage)
	_, hasSize := receivedQuery["size"]
	assert.False(t, hasSize)
	assert.Contains(t, buf.String(), "initial report")
}

// TestOutcomeHistoryEmpty confirms the empty-result path renders "No outcome
// revisions found." in table mode and an empty JSON array in JSON mode.
func TestOutcomeHistoryEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No outcome revisions found.")
}

// TestOutcomeHistoryTeamId mirrors the teamId-query-injection contract used
// elsewhere in this package.
func TestOutcomeHistoryTeamId(t *testing.T) {
	var receivedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("teamId")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "team-456", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newOutcomeHistoryCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"loan-app-1"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "team-456", receivedQuery, "teamId must be present as query parameter")
}
