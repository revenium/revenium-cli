package jobs

import (
	"encoding/json"
	"fmt"

	"github.com/revenium/revenium-cli/internal/output"
)

// undeclared is what a nullable numeric field renders as when the server did
// not declare it. "Not declared" and "declared as zero" are different facts and
// must be distinguishable in the output.
const undeclared = "—"

// money renders a nullable monetary field from a decoded response map.
//
// It is the single numeric-money entry point for this phase, so that the
// economics render and the baselines render both consume it and neither owns
// it.
//
// Two traps it exists to close:
//
//  1. output.FloatVal returns 0 for a missing key, a nil value AND for any
//     value whose type it does not recognise, so an *undeclared* cost — or a
//     cost the server sent in a shape we cannot read — would otherwise render
//     as a confident $0.00. floatAt below keeps "not declared" (—) distinct
//     from "declared as zero" ($0.00), and treats a present-but-unreadable
//     value as the former rather than the latter.
//  2. The unformatted no-verb variant of the fmt print family — which the
//     package-local str() helper uses — emits 1.234567e+06 for a JSON-decoded
//     number at or above 1e6, because encoding/json decodes every JSON number
//     into a float64. str() is therefore for strings and booleans ONLY; every
//     numeric cell goes through money() or num().
func money(m map[string]interface{}, key, currency string) string {
	v, ok := floatAt(m, key)
	if !ok {
		return undeclared
	}
	return output.FormatCurrency(v, currency)
}

// floatAt reads a JSON-decoded numeric value, reporting whether the value is
// one at all.
//
// The second return is what makes the numeric branch TOTAL: absent, null, and
// any non-numeric shape are all reported as "not a number" rather than
// silently becoming zero. A string-encoded decimal is the shape that matters
// in practice — it is a common convention for monetary fields and nothing here
// rules it out, since every schema in play declares `required: []` and
// enforces no types — and rendering "4.25" as $0.00 is a strictly worse
// failure than rendering it as —, because a — prompts a question and a $0.00
// does not.
//
// json.Number is handled because output.FloatVal always handled it; a
// json.Number that does not parse is not a number either.
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

// num renders a nullable non-monetary numeric field using a caller-supplied
// format verb such as "%.2f" (rates) or "%.0f" (counts).
//
// It carries the same two traps as money: a value that is absent, null or not
// a number at all must not render as a confident 0.00, which is what floatAt's
// second return is for; and the verb is mandatory because the unformatted
// variant emits scientific notation at or above 1e6.
func num(m map[string]interface{}, key, verb string) string {
	v, ok := floatAt(m, key)
	if !ok {
		return undeclared
	}
	return fmt.Sprintf(verb, v)
}
