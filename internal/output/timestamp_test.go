package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{"fractional seconds UTC", "2025-10-17T17:53:39.119Z", "2025-10-17 17:53:39 UTC"},
		{"whole seconds UTC", "2024-01-15T10:00:00Z", "2024-01-15 10:00:00 UTC"},
		{"offset normalised to UTC", "2026-01-15T12:00:00+02:00", "2026-01-15 10:00:00 UTC"},
		{"date-only passes through", "2026-01-01", "2026-01-01"},
		{"zone-less passes through", "2025-10-17T17:53:39", "2025-10-17T17:53:39"},
		{"empty stays empty", "", ""},
		{"garbage passes through", "not-a-time", "not-a-time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatTimestamp(tt.raw))
		})
	}
}

func TestTimestampVal(t *testing.T) {
	m := map[string]interface{}{
		"nilValue": nil,
		"created":  "2024-01-15T10:00:00Z",
		"epoch":    float64(1705312800),
		"flag":     true,
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"missing key is empty", "absent", ""},
		{"nil value is empty", "nilValue", ""},
		{"string is formatted", "created", "2024-01-15 10:00:00 UTC"},
		{"float renders like str", "epoch", "1.7053128e+09"},
		{"bool renders like str", "flag", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TimestampVal(m, tt.key))
		})
	}
}
