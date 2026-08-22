package meter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ticketCommands enumerates the four AI metering commands whose schemas define
// ticketId. meter tool-event is deliberately absent: ToolEventMetadataResource
// has no ticketId property, so the flag must not exist there.
func ticketCommands() []struct {
	name     string
	newCmd   func() *cobra.Command
	baseArgs []string
} {
	return []struct {
		name     string
		newCmd   func() *cobra.Command
		baseArgs []string
	}{
		{"completion", newCompletionCmd, requiredCompletionArgs()},
		{"audio", newAudioCmd, []string{
			"--model", "whisper-1",
			"--provider", "openai",
			"--request-time", "2024-01-15T10:00:00Z",
			"--response-time", "2024-01-15T10:00:10Z",
			"--request-duration", "10000",
			"--billing-unit", "PER_SECOND",
		}},
		{"image", newImageCmd, []string{
			"--model", "dall-e-3",
			"--provider", "openai",
			"--request-time", "2024-01-15T10:00:00Z",
			"--response-time", "2024-01-15T10:00:05Z",
			"--request-duration", "5000",
			"--actual-image-count", "1",
			"--billing-unit", "PER_IMAGE",
		}},
		{"video", newVideoCmd, requiredVideoArgs()},
	}
}

// runTicketCmd executes one metering command against an httptest stub and
// returns the request body the CLI actually sent.
func runTicketCmd(t *testing.T, newCmd func() *cobra.Command, args []string) map[string]interface{} {
	t.Helper()

	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": "evt-1", "resourceType": "metered-event", "label": "metered-event", "created": "2024-01-15T10:00:00Z"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCmd()
	c.SetOut(&buf)
	c.SetArgs(args)
	require.NoError(t, c.Execute())

	return receivedBody
}

func TestMeterTicketIDPresentWhenPassed(t *testing.T) {
	for _, tc := range ticketCommands() {
		t.Run(tc.name, func(t *testing.T) {
			body := runTicketCmd(t, tc.newCmd, append(tc.baseArgs, "--ticket-id", "JIRA-123"))
			assert.Equal(t, "JIRA-123", body["ticketId"])
		})
	}
}

// A passed-but-empty --ticket-id must still be sent: the Changed() gate keys on
// whether the operator supplied the flag, not on the value being non-empty.
func TestMeterTicketIDPresentWhenPassedEmpty(t *testing.T) {
	for _, tc := range ticketCommands() {
		t.Run(tc.name, func(t *testing.T) {
			body := runTicketCmd(t, tc.newCmd, append(tc.baseArgs, "--ticket-id", ""))
			require.Contains(t, body, "ticketId")
			assert.Equal(t, "", body["ticketId"])
		})
	}
}

func TestMeterTicketIDAbsentWhenNotPassed(t *testing.T) {
	for _, tc := range ticketCommands() {
		t.Run(tc.name, func(t *testing.T) {
			body := runTicketCmd(t, tc.newCmd, tc.baseArgs)
			assert.NotContains(t, body, "ticketId")
		})
	}
}

// meter tool-event has no ticketId in its schema, so the flag must not be
// registered on it. This guards against a well-meaning copy-paste later.
func TestMeterToolEventHasNoTicketFlag(t *testing.T) {
	c := newToolEventCmd()
	assert.Nil(t, c.Flags().Lookup("ticket-id"))
}
