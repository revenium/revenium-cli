package alerts

import (
	"bytes"
	"encoding/json"
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

func TestBudgetGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/api/ai/alerts/anom-1/budget/progress", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"currentValue": 750.50, "remainingBudget": 249.50, "percentUsed": 75.05, "budgetThreshold": 1000.00, "currency": "USD"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newBudgetGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"anom-1"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "$1,000.00")
	assert.Contains(t, out, "$750.50")
	assert.Contains(t, out, "$249.50")
	assert.Contains(t, out, "75.0%")
}

func TestBudgetGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"currentValue": 750.50, "remainingBudget": 249.50, "percentUsed": 75.05, "budgetThreshold": 1000.00, "currency": "USD"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newBudgetGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"anom-1"})
	err := c.Execute()

	require.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, float64(1000), result["budgetThreshold"])
	assert.Equal(t, "USD", result["currency"])
}
