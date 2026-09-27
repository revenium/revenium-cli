package billing

import (
	"encoding/json"
	"fmt"
)

// undeclared is what a nullable field renders as when the server did not
// declare it. "Not declared" and "declared as zero" are different facts and
// must be distinguishable in the output.
//
// This matters more here than anywhere else in the repo. The seats operation
// description states verbatim that counts come back *absent rather than zero*
// — Anthropic withholds the seat and invite figures for a query scoped to an
// RBAC group — and that "a withheld figure is not an organization that
// assigned no seats". An operator who reads a confident 0 acts on it.
const undeclared = "—"

// floatAt reads a JSON-decoded numeric value, reporting whether the value is
// one at all.
//
// The second return is what makes the numeric branch TOTAL: absent, null, and
// any non-numeric shape are all reported as "not a number" rather than silently
// becoming zero. Six of SeatUtilizationDay_Read's seven properties are declared
// ["integer","null"], so this is the ordinary case for that response, not an
// edge one.
//
// This is deliberately NOT billing.go's floatVal. floatVal returns 0 from its
// default branch for a missing key, a nil value AND any type it does not
// recognise alike, and formatCount then prints that 0 — which asserts a
// measurement that was never taken. floatVal stays correct for the
// non-nullable counts the shipped verbs render; it is simply not usable for a
// nullable column.
//
// json.Number is handled because floatVal always handled it; a json.Number that
// does not parse is not a number either.
func floatAt(m map[string]interface{}, key string) (float64, bool) {
	switch n := m[key].(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// num renders a nullable numeric field using a caller-supplied format verb such
// as "%.0f" (counts) or "%.2f" (rates).
//
// Two traps it exists to close:
//
//  1. A value that is absent, null or not a number at all must not render as a
//     confident 0, which is what floatAt's second return is for.
//  2. The verb is mandatory because the unformatted no-verb variant of the fmt
//     print family — which the package-local str() helper uses — emits
//     1.234567e+06 for a JSON-decoded number at or above 1e6, since
//     encoding/json decodes every JSON number into a float64. str() is
//     therefore for strings and booleans ONLY; every numeric cell goes through
//     num().
func num(m map[string]interface{}, key, verb string) string {
	v, ok := floatAt(m, key)
	if !ok {
		return undeclared
	}
	return fmt.Sprintf(verb, v)
}

// nullableStr renders a nullable string field, keeping an absent or null value
// visually distinct from an empty string the server actually sent.
//
// billing.go's str() collapses both to "", which is the right answer for a
// non-nullable string and the wrong one for a column the schema declares as
// ["string","null"].
func nullableStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return undeclared
	}
	return fmt.Sprint(v)
}

// objectsAt returns the array of objects at key, or nil when the key is absent,
// JSON null, or an empty array.
//
// The nil return is what lets a caller write one `len(...) == 0` empty-state
// branch that covers all three shapes.
func objectsAt(m map[string]interface{}, key string) []map[string]interface{} {
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	items := make([]map[string]interface{}, 0, len(raw))
	for _, entry := range raw {
		if item, ok := entry.(map[string]interface{}); ok {
			items = append(items, item)
		}
	}
	return items
}
