// Package metrics implements the metric query commands for the Revenium CLI.
package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var fromFlag string
var toFlag string

// Cmd is the parent metrics command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "metrics",
	Short: "Query metrics and analytics",
	Example: `  # Query AI metrics for last 24 hours
  revenium metrics ai

  # Query completion metrics with time range
  revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z

  # Query audio metrics as JSON
  revenium metrics audio --json`,
}

func init() {
	Cmd.PersistentFlags().StringVar(&fromFlag, "from", "", "Start date (ISO 8601, e.g. 2024-01-15T00:00:00Z)")
	Cmd.PersistentFlags().StringVar(&toFlag, "to", "", "End date (ISO 8601, e.g. 2024-01-15T23:59:59Z)")

	Cmd.AddCommand(newAICmd())
	Cmd.AddCommand(newCompletionsCmd())
	Cmd.AddCommand(newAudioCmd())
	Cmd.AddCommand(newImageCmd())
	Cmd.AddCommand(newVideoCmd())
	Cmd.AddCommand(newTracesCmd())
	Cmd.AddCommand(newSquadsCmd())
	Cmd.AddCommand(newAPIMetricsCmd())
	Cmd.AddCommand(newToolEventsCmd())
}

// buildPath constructs the API path with time range query parameters.
// When --from and --to are both empty, defaults to last 24 hours.
func buildPath(base string) string {
	from := fromFlag
	to := toFlag

	if from == "" && to == "" {
		now := time.Now().UTC()
		to = now.Format(time.RFC3339)
		from = now.Add(-24 * time.Hour).Format(time.RFC3339)
	}

	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	path := base
	if from != "" {
		path += sep + "startDate=" + url.QueryEscape(from)
		sep = "&"
	}
	if to != "" {
		path += sep + "endDate=" + url.QueryEscape(to)
	}
	return path
}

// formatNumber formats an integer with comma grouping (e.g., 1234567 -> "1,234,567").
func formatNumber(n float64) string {
	intPart := fmt.Sprintf("%.0f", n)
	negative := ""
	if strings.HasPrefix(intPart, "-") {
		negative = "-"
		intPart = intPart[1:]
	}
	if len(intPart) <= 3 {
		return negative + intPart
	}
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return negative + string(result)
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// floatVal safely extracts a float64 from a map, handling float64 and json.Number types.
func floatVal(m map[string]interface{}, key string) float64 {
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
