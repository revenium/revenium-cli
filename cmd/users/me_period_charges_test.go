package users

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

// TestUsersMePeriodCharges is the load-bearing pagination test (05-RESEARCH
// Pitfall 1). It stubs TWO pages of the cursor-based (hasMore/cursor, NOT
// page/totalPages) response shape and asserts a second request follows the
// returned cursor and that items from BOTH pages appear — a naive
// single-page-only implementation would pass a weaker version of this test,
// so the two-page continuation assertion is what matters.
func TestUsersMePeriodCharges(t *testing.T) {
	var pageCharges []string
	var cursorsSeen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/period-charges":
			cursor := r.URL.Query().Get("cursor")
			cursorsSeen = append(cursorsSeen, cursor)
			if cursor == "" {
				pageCharges = append(pageCharges, "page1")
				fmt.Fprint(w, `{
					"_embedded": {"periodChargeResourceList": [{"id":"pc-1","amount":10.5}]},
					"hasMore": true,
					"cursor": "cursor-abc"
				}`)
				return
			}
			if cursor == "cursor-abc" {
				pageCharges = append(pageCharges, "page2")
				fmt.Fprint(w, `{
					"_embedded": {"periodChargeResourceList": [{"id":"pc-2","amount":20.0}]},
					"hasMore": false,
					"cursor": null
				}`)
				return
			}
			t.Fatalf("unexpected cursor value: %q", cursor)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMePeriodChargesCmd()
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

func TestUsersMePeriodChargesEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/api/users/me":
			fmt.Fprint(w, `{"id": "user-1"}`)
		case "/v2/api/users/user-1/period-charges":
			fmt.Fprint(w, `{"_embedded": {"periodChargeResourceList": []}, "hasMore": false, "cursor": null}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newMePeriodChargesCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{})
	err := c.Execute()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No period charges found.")
}

// TestUsersMePeriodChargesNoPageFlags is a regression guard (Pitfall 1):
// this command must NOT expose --page/--page-size, since it is not on the
// standard totalPages-based list path.
func TestUsersMePeriodChargesNoPageFlags(t *testing.T) {
	c := newMePeriodChargesCmd()
	assert.Nil(t, c.Flags().Lookup("page"), "period-charges must not expose --page")
	assert.Nil(t, c.Flags().Lookup("page-size"), "period-charges must not expose --page-size")
}

func TestUsersMePeriodChargesCmdRegistered(t *testing.T) {
	found := false
	for _, c := range meCmd.Commands() {
		if c.Use == "period-charges" {
			found = true
			break
		}
	}
	require.True(t, found, "period-charges subcommand must be registered on the me parent Cmd")
}
