package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

// TestCloneModel verifies that `models clone <id>` with no override flags
// POSTs to /v2/api/sources/ai/models/{id}/clone with an empty-object body.
func TestCloneModel(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedMethod, receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mdl-clone-1","name":"GPT-4 Clone","provider":"OpenAI","mode":"chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCloneCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"global-mdl-1"})

	require.NoError(t, c.Execute())

	assert.Equal(t, "POST", receivedMethod)
	matched, err := regexp.MatchString(`/v2/api/sources/ai/models/.+/clone`, receivedPath)
	require.NoError(t, err)
	assert.True(t, matched, "path %q must match /v2/api/sources/ai/models/{id}/clone", receivedPath)
	assert.Equal(t, "/v2/api/sources/ai/models/global-mdl-1/clone", receivedPath)
	assert.Empty(t, receivedBody, "body must be empty when no override flags are passed")
	assert.Contains(t, buf.String(), "mdl-clone-1")
}

// TestCloneModelOptionalFlags verifies that passing an override flag
// (--input-cost-per-token) adds inputCostPerToken to the body, and other
// override fields stay absent.
func TestCloneModelOptionalFlags(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(bodyBytes, &receivedBody))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mdl-clone-2","name":"GPT-4 Clone","provider":"OpenAI","mode":"chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCloneCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"global-mdl-1", "--input-cost-per-token", "0.00004"})

	require.NoError(t, c.Execute())

	assert.Equal(t, 0.00004, receivedBody["inputCostPerToken"])
	_, hasOutput := receivedBody["outputCostPerToken"]
	assert.False(t, hasOutput, "outputCostPerToken must not be sent when its flag is not passed")
}

// TestCloneModelPathEscape verifies the id is passed through url.PathEscape.
// r.URL.Path is auto-unescaped by net/http, so the escaped form is asserted
// via r.URL.EscapedPath() (the wire-form path actually sent).
func TestCloneModelPathEscape(t *testing.T) {
	var receivedEscapedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedEscapedPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"mdl-clone-3","name":"Clone","provider":"OpenAI","mode":"chat"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCloneCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"global mdl 1"})

	require.NoError(t, c.Execute())
	assert.Equal(t, "/v2/api/sources/ai/models/global%20mdl%201/clone", receivedEscapedPath)
}

// TestCloneModelDryRun pins the dry-run render contract; issues zero HTTP
// calls (no httptest server constructed at all).
func TestCloneModelDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	path := "/v2/api/sources/ai/models/global-mdl-1/clone"
	body := map[string]interface{}{}

	err := dryrun.Render(out, "clone", "model", path, body)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: clone model")
	assert.Contains(t, rendered, path)
	assert.Contains(t, rendered, "No changes were made.")
}

// TestCloneModelMutatingAnnotation guards the mutating annotation contract.
func TestCloneModelMutatingAnnotation(t *testing.T) {
	c := newCloneCmd()
	assert.Equal(t, "true", c.Annotations["mutating"])
}
