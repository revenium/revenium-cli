// Package guardrails — shared filter parser + flag-resolution helpers used by
// budget-rules create and update.
//
// PLAN 260524-kvj decisions (locked):
//   - D-01: --filter and --filters-json are MUTUALLY EXCLUSIVE. The conflict is
//     rejected BEFORE any HTTP call.
//   - D-05: --filter values are split on the FIRST TWO colons only. The value
//     segment may itself contain colons (e.g. "AGENT:IS:my:weird:value" →
//     value="my:weird:value").
//   - D-06: We do NOT validate dimension or operator strings at parse time —
//     they are passed through verbatim and the server is the source of truth.
//     Help text lists the known dimensions/operators as a hint only.
//
// Known operators (hint only — server is authoritative per D-06):
//   - IS, IS_NOT          — exact-match operators
//   - CONTAINS            — value substring match
//   - STARTS_WITH         — value prefix match
//   - ENDS_WITH           — value suffix match
//
// On-wire shape (each filter object):
//
//	{"dimension": "MODEL", "operator": "IS", "value": "gpt-4"}
//	{"dimension": "MODEL", "operator": "CONTAINS", "value": "gpt"}
//	{"dimension": "AGENT", "operator": "STARTS_WITH", "value": "prod-"}
package guardrails

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// parseFilterTriple converts a single "dim:op:val" string into the on-wire
// map shape. Splits on the first two colons only (strings.SplitN with n=3) so
// values containing colons are preserved. All three parts must be non-empty.
//
// The error message includes the raw offending input so users can locate the
// bad flag value in a long invocation.
func parseFilterTriple(s string) (map[string]interface{}, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid --filter value %q: expected dim:op:val (e.g. MODEL:IS:gpt-4)", s)
	}
	dim, op, val := parts[0], parts[1], parts[2]
	if dim == "" || op == "" || val == "" {
		return nil, fmt.Errorf("invalid --filter value %q: dimension, operator, and value must all be non-empty", s)
	}
	// PLAN D-06: pass dim/op through verbatim — server validates the enum.
	// Known operators: IS, IS_NOT, CONTAINS, STARTS_WITH, ENDS_WITH.
	return map[string]interface{}{
		"dimension": dim,
		"operator":  op,
		"value":     val,
	}, nil
}

// parseFiltersJSON unmarshals a --filters-json payload into a slice of
// arbitrary-shape filter maps. We deliberately decode into []map[string]
// interface{} so the caller can pass arbitrary keys through verbatim — the
// server is the source of truth for the filter schema (D-06).
func parseFiltersJSON(s string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("invalid --filters-json: %w", err)
	}
	return out, nil
}

// resolveFilters is the single orchestration helper shared by budget-rules
// create and update. It enforces the D-01 mutual-exclusion contract between
// --filter and --filters-json and returns a (filters, changed, err) triple:
//
//   - filters: parsed slice (nil if neither flag was set)
//   - changed: true iff the user supplied at least one filter source — the
//     caller uses this bool to decide whether to include the "filters" key
//     in the request body (omit when changed=false to avoid empty-array leak)
//   - err: parsing or mutual-exclusion error
//
// The conflict check uses c.Flags().Changed(...) so default values never
// confuse "user passed the flag" with "default empty value".
func resolveFilters(c *cobra.Command, filterFlags []string, filtersJSON string) ([]map[string]interface{}, bool, error) {
	filterSet := c.Flags().Changed("filter")
	jsonSet := c.Flags().Changed("filters-json")

	// PLAN D-01 locked: mutual-exclusion fires BEFORE any HTTP call. The
	// error must name BOTH flags so users know which two to reconcile.
	if filterSet && jsonSet {
		return nil, false, fmt.Errorf("--filter and --filters-json are mutually exclusive; pass at most one")
	}

	if jsonSet {
		parsed, err := parseFiltersJSON(filtersJSON)
		if err != nil {
			return nil, false, err
		}
		return parsed, true, nil
	}

	if filterSet {
		out := make([]map[string]interface{}, 0, len(filterFlags))
		for _, raw := range filterFlags {
			triple, err := parseFilterTriple(raw)
			if err != nil {
				return nil, false, err
			}
			out = append(out, triple)
		}
		return out, true, nil
	}

	// Neither flag set — caller should OMIT the filters key from the body.
	return nil, false, nil
}
