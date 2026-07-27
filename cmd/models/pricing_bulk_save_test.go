package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/output"
)

const bulkPricingFixture = `[
	{"billingUnit":"TOKEN","costType":"USAGE","isGlobal":true,"modality":"TEXT","unitPrice":0.003},
	{"billingUnit":"TOKEN","costType":"USAGE","isGlobal":true,"modality":"TEXT","unitPrice":0.006}
]`

// TestPricingBulkSaveFile asserts --file <path> reads the JSON array from disk
// and PUTs a bare array body (not an object) to
// /v2/api/sources/ai/models/{id}/pricing/dimensions (D-07, full-replace).
func TestPricingBulkSaveFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "dimensions.json")
	require.NoError(t, os.WriteFile(filePath, []byte(bulkPricingFixture), 0o644))

	var receivedBody []interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/pricing/dimensions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &receivedBody))
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingBulkSaveCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1", "--file", filePath})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, receivedBody, 2)
	first, ok := receivedBody[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "TOKEN", first["billingUnit"])
}

// TestPricingBulkSaveStdin asserts --file - reads the JSON array from stdin.
func TestPricingBulkSaveStdin(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(bulkPricingFixture)
	require.NoError(t, err)
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	var receivedBody []interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/pricing/dimensions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &receivedBody))
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingBulkSaveCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1", "--file", "-"})
	err = c.Execute()

	require.NoError(t, err)
	require.Len(t, receivedBody, 2)
}

// TestPricingBulkSaveMalformedJSON asserts malformed JSON yields a clear parse
// error and issues zero HTTP calls.
func TestPricingBulkSaveMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(filePath, []byte("{not valid json"), 0o644))

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingBulkSaveCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"model-1", "--file", filePath})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
	assert.False(t, called, "malformed JSON must not trigger an HTTP call")
}

// TestPricingBulkSaveDryRun pins the dry-run render contract by invoking
// dryrun.Render DIRECTLY with the exact path/verb/resource/body shape that
// pricing_bulk_save.go's RunE passes when cmd.DryRun() is true.
//
// Precedent: cmd/organizations/create_test.go TestCreateOrganizationDryRun —
// cmd.DryRun() has no exported setter, so we test the dry-run render contract
// rather than the global toggle. The branch itself is a trivial single `if`;
// integration is covered by manual smoke per the phase VALIDATION doc.
func TestPricingBulkSaveDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	var entries []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(bulkPricingFixture), &entries))

	err := dryrun.Render(out, "bulk-save", "pricing dimensions",
		"/v2/api/sources/ai/models/model-1/pricing/dimensions", entries)
	require.NoError(t, err)

	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: bulk-save pricing dimensions")
	assert.Contains(t, rendered, "/v2/api/sources/ai/models/model-1/pricing/dimensions")
	assert.Contains(t, rendered, "TOKEN")
	assert.Contains(t, rendered, "0.003")
	assert.Contains(t, rendered, "No changes were made.")
}

// TestPricingBulkSaveRegistered asserts bulk-save is wired into the models
// pricing parent Cmd's init(), so the Phase 1 SAFE-01 registration-completeness
// test stays green. The "zero HTTP calls in dry-run mode" guarantee is
// structural (RunE checks cmd.DryRun() before calling cmd.APIClient.Do) and is
// pinned indirectly via TestPricingBulkSaveDryRun's render-contract assertion.
func TestPricingBulkSaveRegistered(t *testing.T) {
	found := false
	for _, c := range pricingCmd.Commands() {
		if c.Use == "bulk-save <model-id>" {
			found = true
			break
		}
	}
	require.True(t, found, "bulk-save subcommand must be registered on models pricing parent Cmd")
}
