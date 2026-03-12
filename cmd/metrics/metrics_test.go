package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPath_Defaults(t *testing.T) {
	// Reset flags to trigger 24h default
	fromFlag = ""
	toFlag = ""

	path := buildPath("/v2/api/sources/metrics/ai")
	assert.Contains(t, path, "/v2/api/sources/metrics/ai?")
	assert.Contains(t, path, "startDate=")
	assert.Contains(t, path, "endDate=")
}

func TestBuildPath_WithFlags(t *testing.T) {
	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = "2024-01-31T23:59:59Z"
	defer func() { fromFlag = ""; toFlag = "" }()

	path := buildPath("/v2/api/sources/metrics/ai")
	assert.Contains(t, path, "startDate=2024-01-01T00%3A00%3A00Z")
	assert.Contains(t, path, "endDate=2024-01-31T23%3A59%3A59Z")
}

func TestBuildPath_OnlyFrom(t *testing.T) {
	fromFlag = "2024-01-01T00:00:00Z"
	toFlag = ""
	defer func() { fromFlag = "" }()

	path := buildPath("/v2/api/sources/metrics/ai")
	assert.Contains(t, path, "startDate=2024-01-01T00%3A00%3A00Z")
	// Only --from set, no endDate
	assert.False(t, strings.Contains(path, "endDate="))
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	}
	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatNumber(tc.input))
		})
	}
}
