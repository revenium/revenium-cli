package periodcharges

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestPeriodChargesList is the load-bearing pagination test (06-RESEARCH
// Pattern 3 / Pitfall 1). It stubs TWO pages of the cursor-based
// (hasMore/cursor, NOT page/totalPages) response shape and asserts a second
// request follows the returned cursor and that items from BOTH pages
// appear — a naive single-page-only implementation would pass a weaker
// version of this test, so the two-page continuation assertion is what
// matters.
func TestPeriodChargesList(t *testing.T) {
	var pageCharges []string
	var cursorsSeen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/period-charges", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		cursor := r.URL.Query().Get("cursor")
		cursorsSeen = append(cursorsSeen, cursor)
		if cursor == "" {
			pageCharges = append(pageCharges, "page1")
			fmt.Fprint(w, `{
				"_embedded": {"periodChargeResourceList": [{"id":"pc-1","amount":10.5,"currency":"USD"}]},
				"hasMore": true,
				"cursor": "cursor-abc"
			}`)
			return
		}
		if cursor == "cursor-abc" {
			pageCharges = append(pageCharges, "page2")
			fmt.Fprint(w, `{
				"_embedded": {"periodChargeResourceList": [{"id":"pc-2","amount":20.0,"currency":"USD"}]},
				"hasMore": false,
				"cursor": null
			}`)
			return
		}
		t.Fatalf("unexpected cursor value: %q", cursor)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{"page1", "page2"}, pageCharges, "expected exactly two requests, second following the returned cursor")
	require.Equal(t, []string{"", "cursor-abc"}, cursorsSeen)

	out := buf.String()
	assert.Contains(t, out, "pc-1", "item from page 1 must appear")
	assert.Contains(t, out, "pc-2", "item from page 2 must appear")
}

func TestPeriodChargesListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/period-charges", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"periodChargeResourceList": []}, "hasMore": false, "cursor": null}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No period charges found.")
}

// TestPeriodChargesListFilterFlags asserts --invoice-id/--start/--end are
// only added to the query string when explicitly passed, and are mapped to
// the invoiceId/startDate/endDate query param names.
func TestPeriodChargesListFilterFlags(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"_embedded": {"periodChargeResourceList": []}, "hasMore": false, "cursor": null}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newListCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--invoice-id", "inv-123", "--start", "2026-01-01T00:00:00Z", "--end", "2026-02-01T00:00:00Z"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, gotQuery, "invoiceId=inv-123")
	assert.Contains(t, gotQuery, "startDate=2026-01-01T00%3A00%3A00Z")
	assert.Contains(t, gotQuery, "endDate=2026-02-01T00%3A00%3A00Z")
}

// TestPeriodChargesListNoPageFlags is a regression guard (Pitfall 1):
// this command must NOT expose --page/--page-size, since it is not on the
// standard totalPages-based list path.
func TestPeriodChargesListNoPageFlags(t *testing.T) {
	c := newListCmd()
	assert.Nil(t, c.Flags().Lookup("page"), "period-charges list must not expose --page")
	assert.Nil(t, c.Flags().Lookup("page-size"), "period-charges list must not expose --page-size")
}
