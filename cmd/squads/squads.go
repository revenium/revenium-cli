// Package squads implements the Squads API read-side commands for the
// Revenium CLI (list/get/timeline). It is a new top-level resource package
// (D-01) rather than nested under `metrics`, so it can own a `--period`
// enum flag without inheriting the `metrics` parent's `--from`/`--to`
// persistent flags, which don't map to the Squads API's period parameter.
package squads

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// periodFlag holds the --period value shared by every squads subcommand
// (persistent flag). Empty means the flag was not passed — the query param
// is omitted entirely and the server applies its documented THIRTY_DAYS
// default (do not hardcode that default client-side).
var periodFlag string

// Cmd is the parent squads command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "squads",
	Short: "Query multi-agent squad execution and squad-group data",
	Example: `  # List squad groups
  revenium squads list

  # Get a single squad's detail
  revenium squads get squad-loan-proc-12345

  # View a squad's event timeline
  revenium squads timeline squad-loan-proc-12345

  # Scope any of the above to a period
  revenium squads list --period SEVEN_DAYS`,
	PersistentPreRunE: func(c *cobra.Command, args []string) error {
		// Run the root PersistentPreRunE first (config/API client init).
		if root := c.Root(); root != nil && root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(c, args); err != nil {
				return err
			}
		}
		// T-2-02 mitigation: reject unknown --period values before they
		// ever reach the URL (D-01 rationale: squads owns period, not
		// metrics' --from/--to).
		return cmd.ValidatePeriod(periodFlag)
	},
}

func init() {
	Cmd.PersistentFlags().StringVar(&periodFlag, "period", "",
		"Time period: HOUR, EIGHT_HOURS, TWENTY_FOUR_HOURS, SEVEN_DAYS, "+
			"THIRTY_DAYS, NINETY_DAYS, SIX_MONTHS, TWELVE_MONTHS (default THIRTY_DAYS server-side)")
}

// buildSquadsPath appends "?period=..." (or "&period=..." if base already
// has a query string) only when periodFlag is non-empty. Never use
// cmd/metrics' buildPath here — it sends startDate/endDate, and none of the
// five Squads API endpoints accept those params (RESEARCH Pitfall 1).
func buildSquadsPath(base string) string {
	if periodFlag == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "period=" + periodFlag
}

// str safely extracts a string value from a map, returning "" for missing or
// nil keys. Package-local copy (cmd/squads cannot import cmd/metrics'
// unexported helpers) — same duplication tradeoff as cmd/squad_executions.go.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// floatVal safely extracts a float64 from a map, handling float64 and
// json.Number types. Package-local copy.
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

// formatNumber formats an integer with comma grouping (e.g., 1234567 ->
// "1,234,567"). Package-local copy.
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

// formatCost formats a dollar amount with enough precision to show
// significant digits. Package-local copy.
func formatCost(v float64) string {
	if v == 0 {
		return "$0.00"
	}
	if v >= 0.01 || v <= -0.01 {
		return fmt.Sprintf("$%.2f", v)
	}
	s := fmt.Sprintf("$%.7f", v)
	for len(s) > 0 && s[len(s)-1] == '0' {
		trimmed := s[:len(s)-1]
		dot := strings.IndexByte(trimmed, '.')
		if dot >= 0 && len(trimmed)-dot-1 < 2 {
			break
		}
		s = trimmed
	}
	return s
}

// formatDuration formats a millisecond value as a human-readable duration
// string. Package-local copy.
func formatDuration(ms float64) string {
	if ms == 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}
