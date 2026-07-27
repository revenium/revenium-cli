// Package billing implements the read-only billing cost-attribution
// analytics commands for the Revenium CLI (NEW-01). All nine verbs
// (api-keys/daily/models/workspaces/coverage/chart/claude-code-contributions/
// vcs-prs/users) self-register via their own package-local `func init()`
// (squads-style decentralized registration, not the organizations-style
// centralized init() — chosen so each verb file can be added, built, and
// tested independently across Tasks 1-3 without a compile-time dependency
// on verb files that don't exist yet).
package billing

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the parent billing command, exported for registration in main.go
// (deferred to 06-06). All /v2/api/billing/* paths live on the platform
// host with x-api-key auth — do NOT use AnalyticsBaseURL/UseBearerAuth
// (RESEARCH Anti-Patterns).
var Cmd = &cobra.Command{
	Use:   "billing",
	Short: "View billing cost-attribution analytics (read-only)",
	Example: `  # List API-key cost attribution
  revenium billing api-keys

  # Get daily cost/credits breakdown
  revenium billing daily

  # List per-user attributed costs
  revenium billing users

  # Get a specific user's cost detail
  revenium billing users get jane@example.com

  # Get billing API-key data as JSON, including aggregate fields
  revenium billing api-keys --json`,
}

// str safely extracts a string value from a map, returning "" for missing or
// nil keys. Package-local copy (established duplication pattern across
// cmd/organizations, cmd/squads, etc. — see 06-PATTERNS.md).
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// floatVal safely extracts a float64 from a map, handling float64 and
// json.Number types. Package-local copy (cmd/squads/squads.go precedent).
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

// formatCost formats a float64 cost with 2 decimal places and a leading "$".
func formatCost(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}

// formatCount formats a float64 count as a whole-number string.
func formatCount(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

// countStr returns the length of a []interface{} value at key as a string,
// or "0" if the key is absent or not an array. Package-local copy of the
// cmd/anomalies/dimensions.go idiom, reused by the single-object report
// verbs (coverage/chart/claude-code-contributions/vcs-prs).
func countStr(m map[string]interface{}, key string) string {
	if arr, ok := m[key].([]interface{}); ok {
		return fmt.Sprintf("%d", len(arr))
	}
	return "0"
}

// extractEmbeddedItems returns the first array found under
// wrapper["_embedded"] as a slice of maps. Shared unwrap helper for the
// four D-04 aggregate-wrapper billing verbs (api-keys/daily/models/
// workspaces) — see each verb's own D-04 landmine comment for why raw Do +
// this helper replaces DoList.
func extractEmbeddedItems(wrapper map[string]interface{}) []map[string]interface{} {
	var items []map[string]interface{}
	if embedded, ok := wrapper["_embedded"].(map[string]interface{}); ok {
		for _, v := range embedded {
			if arr, ok := v.([]interface{}); ok {
				for _, it := range arr {
					if m, ok := it.(map[string]interface{}); ok {
						items = append(items, m)
					}
				}
				break
			}
		}
	}
	return items
}
