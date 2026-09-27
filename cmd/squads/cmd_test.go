package squads

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestSquadsCmdRejectsInvalidPeriod drives the shipped, package-level Cmd —
// the regression test for the self-recursion defect first found in the copy of
// this package that became cmd/skills (CR-01). Without the `root != Cmd` term
// in Cmd.PersistentPreRunE this does not fail, it CRASHES the test binary with
// `fatal error: stack overflow`, because Cmd is its own root when executed
// unattached and the root hook it delegates to is that same closure.
//
// One test is enough here: cmd/squads already has 18 tests covering its
// behavior. This one exists to hold the guard in place.
func TestSquadsCmdRejectsInvalidPeriod(t *testing.T) {
	prevSilenceErrors := Cmd.SilenceErrors
	prevSilenceUsage := Cmd.SilenceUsage
	Cmd.SilenceErrors = true
	Cmd.SilenceUsage = true
	periodFlag = ""
	t.Cleanup(func() {
		Cmd.SilenceErrors = prevSilenceErrors
		Cmd.SilenceUsage = prevSilenceUsage
		periodFlag = ""
		Cmd.SetArgs(nil)
	})

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSquadEntity)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"list", "--period", "BOGUS"})
	err := Cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Equal(t, 0, requestCount, "invalid --period must fail before any HTTP request")
}

// TestSquadsCmdDelegatesToAttachedRoot is the other half of the `root != Cmd`
// guard, and the direction with the worse failure mode. TestSquadsCmdRejects-
// InvalidPeriod holds the guard against being DROPPED; this holds it against
// being drawn too BROADLY.
//
// When a real root sits above Cmd — as rootCmd does in the shipped binary —
// that root's PersistentPreRunE must still run, because it is what initializes
// config and cmd.APIClient. A guard that skipped it would leave `revenium
// squads list` calling into a nil cmd.APIClient: a nil-pointer panic in the
// shipped binary that no other test in this package observes. Verified by
// no-op'ing the delegation block in Cmd.PersistentPreRunE — before this test
// existed, the entire repository suite stayed green (Phase 22 audit, T-22-10).
//
// SAFE-01 cannot cover this: it tests registration completeness, not whether
// the root hook fires.
func TestSquadsCmdDelegatesToAttachedRoot(t *testing.T) {
	prevSilenceErrors := Cmd.SilenceErrors
	prevSilenceUsage := Cmd.SilenceUsage
	Cmd.SilenceErrors = true
	Cmd.SilenceUsage = true
	periodFlag = ""
	t.Cleanup(func() {
		Cmd.SilenceErrors = prevSilenceErrors
		Cmd.SilenceUsage = prevSilenceUsage
		periodFlag = ""
		Cmd.SetArgs(nil)
	})

	rootRan := false
	parent := &cobra.Command{
		Use:           "revenium",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			rootRan = true
			return nil
		},
	}
	parent.AddCommand(Cmd)
	t.Cleanup(func() { parent.RemoveCommand(Cmd) })

	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetErr(&buf)
	parent.SetArgs([]string{"squads", "list", "--period", "BOGUS"})
	err := parent.Execute()

	assert.True(t, rootRan, "the attached root's PersistentPreRunE must still run")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Contains(t, err.Error(), "not valid")
}
