package skills

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// withRealCmd prepares the package-level Cmd singleton for direct execution
// inside a test and registers the cleanup that undoes it.
//
// Cmd is package state shared by every test in this file, so a test that
// executes it leaves parsed flag values, a Changed bit, and cobra's args
// behind for the next one. This helper silences cobra's own error/usage
// printing (tests assert on the returned error, not on stderr noise), zeroes
// periodFlag, and restores all of it — including Cmd.SetArgs(nil) — on
// cleanup. Every real-Cmd test calls it first.
func withRealCmd(t *testing.T) {
	t.Helper()
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
}

// TestSkillsCmdRejectsInvalidPeriod drives the shipped, package-level Cmd —
// not newListCmd(), not the newPeriodValidatingRoot double — so the T-22-01
// --period gate is proven against the command object an operator (or any
// other Go caller of the exported Cmd) actually runs.
//
// All three assertions are load-bearing. A test that only asserted
// `err != nil` stays green when cmd.ValidatePeriod(periodFlag) is mutated to
// cmd.ValidatePeriod(""): the gate becomes a no-op, `list` runs, and any
// transport hiccup returns a non-nil error for the wrong reason. Asserting
// the message content AND a zero request count is what kills that mutation.
func TestSkillsCmdRejectsInvalidPeriod(t *testing.T) {
	withRealCmd(t)

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
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
	assert.Contains(t, err.Error(), "not valid")
	assert.Equal(t, 0, requestCount, "invalid --period must fail before any HTTP request")
}

// TestSkillsCmdListSendsPeriod proves GAP-01's list command reaches the wire
// through the shipped Cmd: that Cmd's persistent --period flag is genuinely
// bound to periodFlag, and that buildSkillsPath carries it into the query.
// The newPeriodValidatingRoot double re-registers its own flag against the
// same variable, so it could never show this.
func TestSkillsCmdListSendsPeriod(t *testing.T) {
	withRealCmd(t)

	var gotQuery url.Values
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillsPage)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"list", "--period", "SEVEN_DAYS"})

	require.NoError(t, Cmd.Execute())
	assert.Equal(t, 1, requestCount)
	assert.Equal(t, "SEVEN_DAYS", gotQuery.Get("period"))
	assert.Contains(t, buf.String(), "code-review")
}

// TestSkillsCmdGetReachable is GAP-02's half of SC4's "cover both commands":
// the get subcommand is reached through the shipped parent Cmd, not through a
// bare newGetCmd() instance.
func TestSkillsCmdGetReachable(t *testing.T) {
	withRealCmd(t)

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixtureSkillDetail)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	Cmd.SetOut(&buf)
	Cmd.SetErr(&buf)
	Cmd.SetArgs([]string{"get", "JMwX9g4"})

	require.NoError(t, Cmd.Execute())
	assert.Equal(t, "/v2/api/skills/JMwX9g4", gotPath)
	assert.Contains(t, buf.String(), "code-review")
}

// TestSkillsCmdDelegatesToAttachedRoot asserts the OTHER direction of the
// root != Cmd guard: when a real root sits above Cmd — as rootCmd does in the
// shipped binary — that root's PersistentPreRunE must still run, because it is
// what initializes config and cmd.APIClient. A guard drawn too broadly would
// silently ship commands with a nil client.
//
// It also asserts that validation still fires after delegation, which makes
// this the second independent witness to the T-22-01 gate: it dies under the
// same cmd.ValidatePeriod("") mutation that kills
// TestSkillsCmdRejectsInvalidPeriod.
func TestSkillsCmdDelegatesToAttachedRoot(t *testing.T) {
	withRealCmd(t)

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
	parent.SetArgs([]string{"skills", "list", "--period", "BOGUS"})
	err := parent.Execute()

	assert.True(t, rootRan, "the attached root's PersistentPreRunE must still run")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Contains(t, err.Error(), "not valid")
}

// TestSkillsCmdWiring asserts Cmd's registration properties directly against
// the singleton — including the SC3 read-only claim, which until now existed
// only as a code comment.
func TestSkillsCmdWiring(t *testing.T) {
	withRealCmd(t)

	assert.NotNil(t, Cmd.PersistentFlags().Lookup("period"),
		"--period is a persistent flag on Cmd, shared by every subcommand")
	assert.Empty(t, Cmd.Annotations["mutating"],
		"both skills subcommands are read-only (SC3)")

	var names []string
	for _, sub := range Cmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "get")
}
