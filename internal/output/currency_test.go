package output

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		expected string
	}{
		{"zero USD", 0, "USD", "$0.00"},
		{"small amount", 5.99, "USD", "$5.99"},
		{"thousand", 1000.00, "USD", "$1,000.00"},
		{"million", 1000000.50, "USD", "$1,000,000.50"},
		{"negative", -500.00, "USD", "-$500.00"},
		{"negative large", -1500.75, "USD", "-$1,500.75"},
		{"empty currency defaults to dollar", 100.00, "", "$100.00"},
		{"non-USD currency", 1000.00, "EUR", "EUR 1,000.00"},
		{"GBP currency", 2500.50, "GBP", "GBP 2,500.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCurrency(tt.amount, tt.currency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFloatVal(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected float64
	}{
		{"float64 value", map[string]interface{}{"k": 1.5}, "k", 1.5},
		{"json.Number value", map[string]interface{}{"k": json.Number("2.5")}, "k", 2.5},
		{"missing key", map[string]interface{}{"other": 1}, "missing", 0},
		{"nil value", map[string]interface{}{"k": nil}, "k", 0},
		{"unhandled string type", map[string]interface{}{"k": "string-not-number"}, "k", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FloatVal(tt.input, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
