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
	// Assigned here rather than inside the composite literal above: naming Cmd
	// within its own initializer is the Go initialization cycle
	// ("initialization cycle: Cmd refers to itself").
	//
	// The root PersistentPreRunE has to run first because it is what
	// initializes config and cmd.APIClient; skipping it would leave the
	// shipped `revenium metrics ...` commands calling into a nil client.
	//
	// Why the guard compares root against Cmd and not against c: cobra walks
	// up from the executed command to the nearest ancestor carrying a
	// PersistentPreRunE, then invokes that hook passing the executed LEAF as
	// c. Running Cmd unattached with `ai --from X` therefore enters here with
	// c == the ai command and root == Cmd, so the leaf form would never trip
	// and this closure would call itself until the process dies with
	// `fatal error: stack overflow`. Comparing against the hook's OWNER is
	// what actually detects "Cmd is its own root", i.e. that the root hook we
	// would delegate to IS this very closure.
	//
	// cmd/metrics/dimensions.go carries the leaf-command variant of this same
	// guard (comparing root against c); that form is correct there precisely
	// because dimensions is a leaf, so there c IS the hook's owner. The
	// divergence is deliberate, not drift.
	Cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		// Run the root PersistentPreRunE first (config/API client init) —
		// but only when a real root sits above Cmd, as it does under
		// main.go's rootCmd.
		if root := c.Root(); root != nil && root != Cmd && root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(c, args); err != nil {
				return err
			}
		}
		if err := normalizeDateFlag("from", &fromFlag); err != nil {
			return err
		}
		return normalizeDateFlag("to", &toFlag)
	}

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
	Cmd.AddCommand(newDimensionsCmd())
}

// normalizeDateFlag parses a date string, appending "Z" if no timezone is present,
// and stores the normalized value back into the flag variable.
func normalizeDateFlag(name string, flag *string) error {
	if *flag == "" {
		return nil
	}
	// Already valid RFC 3339
	if _, err := time.Parse(time.RFC3339, *flag); err == nil {
		return nil
	}
	// Try appending Z for inputs like "2025-01-01T00:00:00"
	withZ := *flag + "Z"
	if _, err := time.Parse(time.RFC3339, withZ); err == nil {
		*flag = withZ
		return nil
	}
	return fmt.Errorf("--%s %q is not valid ISO 8601 format (expected e.g. 2025-01-01T00:00:00Z)", name, *flag)
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

// formatCost formats a dollar amount with enough precision to show significant digits.
// Values >= $0.01 use 2 decimals, otherwise up to 7 decimals with trailing zeros trimmed.
func formatCost(v float64) string {
	if v == 0 {
		return "$0.00"
	}
	if v >= 0.01 || v <= -0.01 {
		return fmt.Sprintf("$%.2f", v)
	}
	s := fmt.Sprintf("$%.7f", v)
	// Trim trailing zeros but keep at least 2 decimal places
	for len(s) > 0 && s[len(s)-1] == '0' {
		trimmed := s[:len(s)-1]
		// Count decimals remaining
		dot := strings.IndexByte(trimmed, '.')
		if dot >= 0 && len(trimmed)-dot-1 < 2 {
			break
		}
		s = trimmed
	}
	return s
}

// formatDuration formats a millisecond value as a human-readable duration string.
// Values under 1000ms show as "123ms", otherwise as "1.23s".
func formatDuration(ms float64) string {
	if ms == 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// nestedStr extracts a string field from a nested object.
// For example, nestedStr(m, "organization", "label") returns m["organization"]["label"].
func nestedStr(m map[string]interface{}, objKey, field string) string {
	if obj, ok := m[objKey].(map[string]interface{}); ok {
		return str(obj, field)
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

// squadCell renders the Squad column: squadName if present, else squadId,
// else blank (D-06/D-07). Safe to call even on schemas (ai/audio/video/traces)
// that never populate squadName — str() returns "" for a missing key, so the
// fallback to squadId still fires correctly.
func squadCell(m map[string]interface{}) string {
	if name := str(m, "squadName"); name != "" {
		return name
	}
	return str(m, "squadId")
}

// filterBySquadID returns only the rows whose squadId field exactly matches id.
// Client-side filter (D-09) — no server-side squadId query param exists on any
// of the five general metrics endpoints. Zero matches is a normal empty result,
// not an error (D-10).
func filterBySquadID(metrics []map[string]interface{}, id string) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0, len(metrics))
	for _, m := range metrics {
		if str(m, "squadId") == id {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
