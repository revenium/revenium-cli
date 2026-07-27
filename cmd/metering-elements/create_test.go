package meteringelements

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestMeteringElementsCreate verifies that
// `metering-elements create --name X --type NUMBER` POSTs to
// /v2/api/metering-element-definitions with a body containing only
// "name"/"type" — proving the optional --description field is omitted when
// not supplied.
func TestMeteringElementsCreate(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/metering-element-definitions", r.URL.Path)
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		assert.Equal(t, "Tokens", receivedBody["name"])
		assert.Equal(t, "NUMBER", receivedBody["type"])
		_, hasDescription := receivedBody["description"]
		assert.False(t, hasDescription, "description must not be sent when --description not provided")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-new","name":"Tokens","type":"NUMBER","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "Tokens", "--type", "NUMBER"})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "me-new")
}

// TestMeteringElementsCreateOptionalFields is the inverse: when
// --description IS passed, it must appear in the request body.
func TestMeteringElementsCreateOptionalFields(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))
		assert.Equal(t, "Token count", receivedBody["description"])
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"me-new2","name":"Tokens","type":"NUMBER","description":"Token count","_links":{}}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--name", "Tokens", "--type", "NUMBER", "--description", "Token count"})

	require.NoError(t, c.Execute())
	assert.Contains(t, buf.String(), "me-new2")
}

// TestMeteringElementsCreateInvalidType is the LOAD-BEARING T-06-13
// assertion: an invalid --type value must be rejected client-side with NO
// HTTP request issued. Configures an unreachable BaseURL so an accidental
// HTTP attempt surfaces as a connection error rather than a validation one.
func TestMeteringElementsCreateInvalidType(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--name", "Tokens", "--type", "BOGUS"})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOGUS")
	assert.Contains(t, err.Error(), "STRING or NUMBER")
}

// TestMeteringElementsCreateMissingRequired asserts omitting --name or
// --type errors at the Cobra MarkFlagRequired layer before any HTTP call.
func TestMeteringElementsCreateMissingRequired(t *testing.T) {
	var buf bytes.Buffer
	cmd.APIClient = api.NewClient("http://unused", "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--type", "NUMBER"})

	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

// TestMeteringElementsCreateDryRun pins the dry-run output contract by
// invoking dryrun.Render directly with the exact shape create.go's RunE
// passes when cmd.DryRun() is true.
func TestMeteringElementsCreateDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	body := map[string]interface{}{
		"name": "Tokens",
		"type": "NUMBER",
	}

	err := dryrun.Render(out, "create", "metering element", "/v2/api/metering-element-definitions", body)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: create metering element")
	assert.Contains(t, rendered, "/v2/api/metering-element-definitions")
	assert.Contains(t, rendered, "No changes were made.")
}
