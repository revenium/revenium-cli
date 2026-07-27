package metrics

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIMetrics asserts the dead-endpoint error path (D-06): no HTTP server
// is constructed in this test, so a passing test structurally proves zero
// HTTP calls are made by `metrics api`.
func TestAPIMetrics(t *testing.T) {
	var buf bytes.Buffer

	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"

	c := newAPIMetricsCmd()
	c.SetOut(&buf)
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not currently available")
	assert.Contains(t, err.Error(), "docs/analytics-coverage.md")
}
