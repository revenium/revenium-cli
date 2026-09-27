package jobs

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeBody decodes a JSON object body exactly the way api.Client.Do does, so
// the helpers under test see the same float64 values the real code path sees.
func decodeBody(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &m))
	return m
}

// TestMoneyRendersAbsentAndNull pins the distinction money() exists to make:
// an undeclared value and a value declared as zero are different facts and
// must be provably distinct in the output.
//
// Mutation this must fail under: remove the presence branch from money().
// output.FloatVal returns 0 for both a missing key and an explicit null, so
// all three cases would collapse to "$0.00".
func TestMoneyRendersAbsentAndNull(t *testing.T) {
	absent := decodeBody(t, `{}`)
	assert.Equal(t, "—", money(absent, "overheadPerUnit", "USD"),
		"an absent key means not declared, not zero")

	null := decodeBody(t, `{"overheadPerUnit": null}`)
	assert.Equal(t, "—", money(null, "overheadPerUnit", "USD"),
		"an explicit JSON null means not declared, not zero")

	zero := decodeBody(t, `{"overheadPerUnit": 0}`)
	assert.Equal(t, "$0.00", money(zero, "overheadPerUnit", "USD"),
		"a declared zero must render as a confident $0.00")
}

// TestMoneyRendersLargeValue pins the scientific-notation trap. encoding/json
// decodes every JSON number into a float64, and the unformatted print variant
// the package-local str() helper uses renders 1234567 as "1.234567e+06".
//
// Mutation this must fail under: swap money() back to str().
func TestMoneyRendersLargeValue(t *testing.T) {
	m := decodeBody(t, `{"overheadPerUnit": 1234567}`)
	assert.Equal(t, "$1,234,567.00", money(m, "overheadPerUnit", "USD"))
}

// TestNumRendersAbsentAndNull covers the same presence branch for the
// non-monetary helper, which plan 27-03's baseline render consumes for rates
// and counts. Without it the helper ships unpinned.
func TestNumRendersAbsentAndNull(t *testing.T) {
	absent := decodeBody(t, `{}`)
	assert.Equal(t, "—", num(absent, "qualityRate", "%.2f"))

	null := decodeBody(t, `{"qualityRate": null}`)
	assert.Equal(t, "—", num(null, "qualityRate", "%.2f"))

	zero := decodeBody(t, `{"qualityRate": 0}`)
	assert.Equal(t, "0.00", num(zero, "qualityRate", "%.2f"))
}

// TestNumRendersLargeValue pins that num() never emits scientific notation
// either — the same trap, on the counts-and-rates path.
func TestNumRendersLargeValue(t *testing.T) {
	m := decodeBody(t, `{"minutesPerUnit": 12345678901}`)
	assert.Equal(t, "12345678901", num(m, "minutesPerUnit", "%.0f"))
}

// decodeBodyUsingNumbers decodes with UseNumber, so the helpers see
// json.Number rather than float64. api.Client.Do does not currently set it,
// but output.FloatVal has always handled the type and the shape check added
// for WR-05 must not quietly drop that support.
func decodeBodyUsingNumbers(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var m map[string]interface{}
	require.NoError(t, dec.Decode(&m))
	return m
}

// TestMoneyRejectsNonNumericShapes pins WR-05.
//
// The presence branch only asked whether the key existed and was not nil; the
// value then went to output.FloatVal, which understands float64 and
// json.Number and returns 0 for everything else. A string-encoded decimal — a
// common convention for monetary fields, and one nothing in this codebase
// rules out, since every schema in play declares `required: []` and enforces
// no types — therefore rendered as a confident $0.00.
//
// That is strictly worse than the absent case the helper already handled: a
// "—" prompts a question and a "$0.00" does not. $0.00 for a pre-AI cost of
// $4.25 is precisely the wrong-but-plausible number money()'s doc comment says
// it exists to eliminate, and it reaches both the `economics get` baseline
// block and every `baselines list` row.
//
// Mutation this must fail under: make the shape check fall back to
// output.FloatVal for an unrecognised type.
func TestMoneyRejectsNonNumericShapes(t *testing.T) {
	t.Run("a string-encoded decimal is not a number", func(t *testing.T) {
		m := decodeBody(t, `{"costPerUnit": "4.25"}`)
		assert.Equal(t, "—", money(m, "costPerUnit", "USD"),
			"$0.00 for a declared $4.25 is the wrong-but-plausible number money() exists to prevent")
	})

	t.Run("every other non-numeric shape", func(t *testing.T) {
		for name, body := range map[string]string{
			"boolean": `{"costPerUnit": true}`,
			"object":  `{"costPerUnit": {"amount": 4.25}}`,
			"array":   `{"costPerUnit": [4.25]}`,
			"string":  `{"costPerUnit": "not a number at all"}`,
		} {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, "—", money(decodeBody(t, body), "costPerUnit", "USD"))
			})
		}
	})

	t.Run("num carries the same check", func(t *testing.T) {
		m := decodeBody(t, `{"qualityRate": "0.9"}`)
		assert.Equal(t, "—", num(m, "qualityRate", "%.2f"),
			"a present but unreadable rate must not render as 0.00")
	})

	t.Run("a genuine number is still a number", func(t *testing.T) {
		m := decodeBody(t, `{"costPerUnit": 4.25, "qualityRate": 0.9}`)
		assert.Equal(t, "$4.25", money(m, "costPerUnit", "USD"))
		assert.Equal(t, "0.90", num(m, "qualityRate", "%.2f"))
	})

	t.Run("json.Number is still a number", func(t *testing.T) {
		m := decodeBodyUsingNumbers(t, `{"costPerUnit": 1234567, "qualityRate": 0.9}`)
		assert.Equal(t, "$1,234,567.00", money(m, "costPerUnit", "USD"),
			"the json.Number path must survive the shape check, and without scientific notation")
		assert.Equal(t, "0.90", num(m, "qualityRate", "%.2f"))
	})
}
