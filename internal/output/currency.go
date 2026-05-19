package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatCurrency formats a number as currency with commas and 2 decimal places.
// Uses "$" prefix for USD or empty currency. Other currencies use code prefix (e.g., "EUR 1,000.00").
func FormatCurrency(amount float64, currency string) string {
	formatted := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	negative := ""
	if strings.HasPrefix(intPart, "-") {
		negative = "-"
		intPart = intPart[1:]
	}
	if len(intPart) > 3 {
		var result []byte
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result = append(result, ',')
			}
			result = append(result, byte(c))
		}
		intPart = string(result)
	}
	symbol := "$"
	if currency != "" && currency != "USD" {
		symbol = currency + " "
	}
	return fmt.Sprintf("%s%s%s.%s", negative, symbol, intPart, parts[1])
}

// FloatVal safely extracts a float64 from a map, handling float64 and json.Number types.
// Returns 0 for missing key, nil value, or unhandled type.
func FloatVal(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return 0
}
