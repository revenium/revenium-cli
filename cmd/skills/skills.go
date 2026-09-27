// Package skills implements the read side of the v1.4.0
// `meter completion --skill-*` write path: the commands that read back the
// skill usage and cost data the CLI has been writing since v1.4.0, closing
// the write-without-read asymmetry that shipped with it (GAP-01, GAP-02).
//
// Both operations (listSkills, getSkillDetail) live on the PLATFORM API
// document, not metering and not analytics (D-22-01). This package therefore
// uses cmd.APIClient exactly as configured and deliberately makes NO base-URL
// change: no MeterBaseURL(), no AnalyticsBaseURL, no bearer auth, and no
// PersistentPreRunE base-URL mutation. Adding one would silently retarget
// these read commands at a surface that does not serve them.
//
// The package mirrors cmd/squads structurally — a new top-level resource
// package owning its own `--period` enum flag rather than nesting under
// `metrics`, whose `--from`/`--to` persistent flags do not map to the
// platform API's period parameter.
package skills

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// periodFlag holds the --period value shared by every skills subcommand
// (persistent flag). Empty means the flag was not passed — the query param
// is omitted entirely and the server applies its documented THIRTY_DAYS
// default. Do not hardcode that default client-side (D-22-04): the spec
// documents it in prose but declares no `default:` in the parameter schema,
// and baking it in is a recorded v1.3 anti-pattern.
var periodFlag string

// Cmd is the parent skills command, exported for registration in main.go.
// It carries no Annotations{"mutating": ...} — both subcommands are
// read-only (SC3).
var Cmd = &cobra.Command{
	Use:   "skills",
	Short: "Query Claude Code skill usage and attributed cost",
	Example: `  # List skills by attributed cost
  revenium skills list

  # Get a single skill's usage detail
  revenium skills get JMwX9g4

  # Scope either of the above to a period
  revenium skills list --period SEVEN_DAYS

  # Emit full-fidelity JSON instead of a table
  revenium skills list --json`,
}

func init() {
	// The hook is assigned here rather than as a field of the Cmd composite
	// literal above because its guard has to name Cmd, and referring to Cmd
	// from inside its own initializer is a Go initialization cycle
	// ("initialization cycle: Cmd refers to itself").
	//
	// Why the guard compares root against Cmd and not against c: cobra walks
	// up from the executed command to the nearest ancestor carrying a
	// PersistentPreRunE, then invokes that hook passing the executed LEAF as
	// c. Running Cmd unattached with `list --period X` therefore enters here
	// with c == the list command and root == Cmd, so the `root != c` form
	// never trips and this closure calls itself until the process dies with
	// `fatal error: stack overflow`. Comparing against the hook's owner —
	// root != Cmd — is what actually detects "Cmd is its own root", i.e. that
	// the root hook we would delegate to IS this very closure.
	//
	// cmd/metrics/dimensions.go carries the leaf-command variant of this same
	// guard (`root != c`); that form is correct there precisely because
	// dimensions is a leaf, so c IS the hook's owner. The divergence here is
	// deliberate, not drift.
	Cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		// Run the root PersistentPreRunE first (config/API client init) —
		// but only when a real root sits above Cmd, as it does under
		// main.go's rootCmd.
		if root := c.Root(); root != nil && root != Cmd && root.PersistentPreRunE != nil {
			if err := root.PersistentPreRunE(c, args); err != nil {
				return err
			}
		}
		// T-22-01 mitigation: reject unknown --period values before they
		// are ever interpolated into a request URL.
		return cmd.ValidatePeriod(periodFlag)
	}

	Cmd.PersistentFlags().StringVar(&periodFlag, "period", "",
		"Time period: HOUR, EIGHT_HOURS, TWENTY_FOUR_HOURS, SEVEN_DAYS, "+
			"THIRTY_DAYS, NINETY_DAYS, SIX_MONTHS, TWELVE_MONTHS (default THIRTY_DAYS server-side)")
}

// buildSkillsPath appends "?period=..." (or "&period=..." if base already has
// a query string) only when periodFlag is non-empty.
//
// The literal endpoint path must stay the FIRST argument at every call site:
// the Phase 20 coverage-audit extractor resolves a local helper through its
// first argument beginning with "/", and that is what keeps these call sites
// resolvable to their spec operations.
func buildSkillsPath(base string) string {
	if periodFlag == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "period=" + periodFlag
}

// The four helpers below (str, floatVal, formatNumber, formatCost) are a
// package-local copy — the third in the repository, after cmd/metrics and
// cmd/squads. A shared home for them is a known and wanted future
// consolidation, but promoting them now would mean editing two packages this
// phase does not otherwise touch, so the duplication is taken deliberately
// rather than by omission. Only these four are copied: an unused
// package-level function trips golangci-lint's `unused` checker.

// str safely extracts a string value from a map, returning "" for missing or
// nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// floatVal safely extracts a float64 from a map, handling float64 and
// json.Number types.
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
// "1,234,567").
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
// significant digits.
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
