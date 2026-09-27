package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
)

// validPeriods is the exact eight-value `period` enum shared by the
// /v2/api/squads* endpoints and the /v2/api/skills endpoints (HOUR,
// EIGHT_HOURS, TWENTY_FOUR_HOURS, SEVEN_DAYS, THIRTY_DAYS, NINETY_DAYS,
// SIX_MONTHS, TWELVE_MONTHS), verified value-for-value against the cached
// platform spec on 2026-08-23.
//
// Every one of those endpoints documents a THIRTY_DAYS server-side default
// when period is omitted — do not hardcode that default client-side (a
// recorded v1.3 anti-pattern). An empty string means the caller omits the
// query param entirely and lets the server apply it.
//
// Two caller families reach these values through ValidatePeriod: cmd/squads
// (list, get, timeline, executions) and cmd/skills (list, get). A third
// surface adopting a `period` parameter should join them here rather than
// declaring a second copy — one enum, not two (D-22-02). Two copies of one
// enum is precisely the drift this milestone exists to detect.
var validPeriods = []string{
	"HOUR", "EIGHT_HOURS", "TWENTY_FOUR_HOURS", "SEVEN_DAYS",
	"THIRTY_DAYS", "NINETY_DAYS", "SIX_MONTHS", "TWELVE_MONTHS",
}

// ValidatePeriod validates p against the shared platform `period` enum
// declared above. It is called by both the squads and the skills command
// families, so its wording and error message stay surface-neutral.
//
// An empty string is valid — it means the caller omits the query param
// entirely, letting the server apply its documented THIRTY_DAYS default
// (T-2-02 / T-22-01 mitigation: reject unknown values before they are ever
// interpolated into a request URL).
func ValidatePeriod(p string) error {
	if p == "" {
		return nil
	}
	for _, v := range validPeriods {
		if p == v {
			return nil
		}
	}
	return fmt.Errorf("--period %q is not valid (expected one of: %s)", p, strings.Join(validPeriods, ", "))
}

// SquadExecutionsTableDef defines the table layout for flat squad-execution
// rows (D-04), shared by `cmd/squads executions` (no-arg) and the rewritten
// `cmd/metrics squads` (deprecated).
var SquadExecutionsTableDef = output.TableDef{
	Headers:      []string{"ID", "Squad", "Agents", "Traces", "Duration", "Cost", "Status"},
	StatusColumn: 6,
}

// FetchSquadExecutions calls GET /v2/api/squads (flat executions list) and
// returns the raw metrics. Both `cmd/squads executions` (no positional arg)
// and the deprecated `cmd/metrics squads` call this so the fetch+render
// logic is never duplicated (D-04). period is appended as a query param only
// when non-empty; an empty period omits the param entirely so the server
// applies its documented THIRTY_DAYS default.
func FetchSquadExecutions(ctx context.Context, period string) ([]map[string]interface{}, error) {
	path := "/v2/api/squads"
	if period != "" {
		path += "?period=" + period
	}
	var metrics []map[string]interface{}
	err := APIClient.DoList(ctx, path, api.ListOptions{FetchAll: !Output.IsJSON()}, &metrics)
	return metrics, err
}

// ToSquadExecutionRows maps the real EntityModelSquadExecutionResource_Read
// fields (id, squadName, agentCount, traceCount, duration, totalCost,
// status) to table rows — never the fictional transactionId/name/executions
// fields the previous `metrics squads` implementation read.
func ToSquadExecutionRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "id"), str(m, "squadName"),
			formatNumber(floatVal(m, "agentCount")), formatNumber(floatVal(m, "traceCount")),
			formatDuration(floatVal(m, "duration")), formatCost(floatVal(m, "totalCost")),
			str(m, "status"),
		}
	}
	return rows
}

// str safely extracts a string value from a map, returning "" for missing or
// nil keys. Duplicated from cmd/metrics/metrics.go's unexported helper of
// the same name (RESEARCH's minimal-duplication option) since package cmd
// cannot import cmd/metrics (would create the exact import cycle this
// shared-helper file exists to avoid).
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// floatVal safely extracts a float64 from a map, handling float64 and
// json.Number types. Duplicated from cmd/metrics/metrics.go.
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
// "1,234,567"). Duplicated from cmd/metrics/metrics.go.
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
// significant digits. Values >= $0.01 use 2 decimals, otherwise up to 7
// decimals with trailing zeros trimmed. Duplicated from
// cmd/metrics/metrics.go.
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
// string. Values under 1000ms show as "123ms", otherwise as "1.23s".
// Duplicated from cmd/metrics/metrics.go.
func formatDuration(ms float64) string {
	if ms == 0 {
		return ""
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}
