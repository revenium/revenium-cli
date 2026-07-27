package models

import (
	"bytes"
	"encoding/json"
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

// TestPricingCoverage asserts the coverage verb hits the LOCKED
// /v2/api/sources/ai/models/{id}/pricing/coverage path (RES-02) and renders
// all 6 boolean fields of PricingCoverageResource_Read as a key-value table.
func TestPricingCoverage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/pricing/coverage")
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/pricing/coverage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"hasAudioPricing": true,
			"hasCharacterPricing": false,
			"hasCreditsPricing": true,
			"hasImagePricing": false,
			"hasTokenPricing": true,
			"hasVideoPricing": false,
			"_links": {}
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingCoverageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "hasAudioPricing")
	assert.Contains(t, out, "hasCharacterPricing")
	assert.Contains(t, out, "hasCreditsPricing")
	assert.Contains(t, out, "hasImagePricing")
	assert.Contains(t, out, "hasTokenPricing")
	assert.Contains(t, out, "hasVideoPricing")
	assert.NotContains(t, out, "_links")
}

// TestPricingCoverageJSON asserts JSON mode passes the raw response through.
func TestPricingCoverageJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/sources/ai/models/model-1/pricing/coverage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"hasAudioPricing": true,
			"hasCharacterPricing": false,
			"hasCreditsPricing": true,
			"hasImagePricing": false,
			"hasTokenPricing": true,
			"hasVideoPricing": false
		}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newPricingCoverageCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"model-1"})
	err := c.Execute()

	require.NoError(t, err)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, true, parsed["hasAudioPricing"])
	assert.Equal(t, false, parsed["hasCharacterPricing"])
}

// TestPricingCoverageRegistered asserts coverage is wired into the models
// pricing parent Cmd's init(), so the Phase 1 SAFE-01 registration-completeness
// test stays green.
func TestPricingCoverageRegistered(t *testing.T) {
	found := false
	for _, c := range pricingCmd.Commands() {
		if c.Use == "coverage <model-id>" {
			found = true
			break
		}
	}
	require.True(t, found, "coverage subcommand must be registered on models pricing parent Cmd")
}
