package jobs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file pins the PURE stages of the facts pipeline — the reader and the
// validator. There is no HTTP anywhere in it and no command is executed: every
// question these tests ask is decidable without a server, which is what makes
// them the cheapest place to pin the phase's refusal rules. The command-level
// counterparts, which assert that a refusal issues ZERO requests, live in
// facts_append_test.go.
//
// Fixtures are decoded with decodeFactEntries (facts_append_test.go) rather
// than hand-built as Go literals, so what a test declares is the JSON an
// operator would actually write.

// TestFactProvenanceValuesAreTheFourFactValues pins the set itself, not merely
// a behaviour that happens to use it.
//
// PeriodFactEntry.properties.provenance.enum declares FOUR values. Baselines'
// enum declares three, and MEASURED is the only member the two share — so a
// fifth value slipping in here, or the set being quietly swapped for the
// baseline one, is the mutation this test exists to catch, ahead of any
// message-text assertion.
func TestFactProvenanceValuesAreTheFourFactValues(t *testing.T) {
	assert.Equal(t,
		[]string{"MEASURED", "SELF_REPORTED", "DERIVED", "ATTESTED"},
		factProvenanceValues,
		"exactly the four values PeriodFactEntry declares, in schema order")
	require.Len(t, factProvenanceValues, 4, "four, not three and not five")
}

// TestValidateFactEntriesRejectsProvenance is JOBS-15 SC2's message contract:
// the refusal must name the offending ARRAY INDEX, the value that was supplied
// and every value that would have been accepted.
//
// The index matters because a file carrying a hundred entries is unfixable
// from "invalid provenance"; the legal values matter because an operator who
// mistyped should not have to open the API spec to find out what was expected.
func TestValidateFactEntriesRejectsProvenance(t *testing.T) {
	entries := decodeFactEntries(t, `[
	  {"key":"documents_reviewed","value":10},
	  {"key":"documents_reviewed","value":11},
	  {"key":"documents_reviewed","value":12},
	  {"key":"documents_reviewed","value":13,"provenance":"MEASUED"}
	]`)

	err := validateFactEntries(entries)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry [3].provenance",
		"the refusal must name the position in the array the operator supplied")
	assert.Contains(t, err.Error(), "MEASUED", "the offending value must be quoted back")
	for _, legal := range factProvenanceValues {
		assert.Contains(t, err.Error(), legal,
			"every legal value must be named — the operator should not need the spec")
	}
}

// TestValidateFactEntriesRejectsBaselineProvenance is the mirror image of the
// pin baselines_append.go already carries from the other side.
//
// The two enums overlap in exactly one member. A validator that reached for
// baselineProvenanceValues by mistake would still accept MEASURED and would
// still reject a typo, so only the two baseline-ONLY values and the one
// fact-only value can tell the two sets apart. That is what this test asserts.
func TestValidateFactEntriesRejectsBaselineProvenance(t *testing.T) {
	for _, baselineOnly := range []string{"CUSTOMER_DECLARED", "SIGNED_OFF"} {
		t.Run(baselineOnly, func(t *testing.T) {
			entries := decodeFactEntries(t,
				`[{"key":"documents_reviewed","value":1,"provenance":"`+baselineOnly+`"}]`)
			err := validateFactEntries(entries)
			require.Error(t, err,
				"%s is a BASELINE provenance and means nothing on a period fact", baselineOnly)
			assert.Contains(t, err.Error(), "entry [0].provenance")
		})
	}

	t.Run("SELF_REPORTED", func(t *testing.T) {
		entries := decodeFactEntries(t,
			`[{"key":"documents_reviewed","value":1,"provenance":"SELF_REPORTED"}]`)
		assert.NoError(t, validateFactEntries(entries),
			"SELF_REPORTED is a fact-only value and must be accepted here")
	})
}

// TestValidateFactEntriesRateBound pins D-28-05: the 0-to-1 bound is keyed on
// the literal metric key `quality_rate`, and it is INCLUSIVE at both ends.
//
// The last case is the load-bearing one. Without a 1.5 under a DIFFERENT key,
// a mutation that bounded every numeric value in the file would stay green —
// and that mutation would refuse a legitimate percentage or count, which is
// precisely the "refuse input the API accepts" failure D-27-11 named.
//
// Inclusivity is an assumption carried openly: the schema states the rule in
// prose with no minimum/maximum keyword to settle it, and refusing a
// legitimately perfect rate of exactly 1 would be the worse of the two errors.
func TestValidateFactEntriesRateBound(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{"above one is refused", "quality_rate", "1.5", true},
		{"below zero is refused", "quality_rate", "-0.1", true},
		{"zero is a rate", "quality_rate", "0", false},
		{"one is a rate", "quality_rate", "1", false},
		{"a half is a rate", "quality_rate", "0.5", false},
		{"the bound is keyed on the metric key, not on the value", "documents_reviewed", "1.5", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries := decodeFactEntries(t,
				`[{"key":"`+tc.key+`","value":`+tc.value+`}]`)
			err := validateFactEntries(entries)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "entry [0].value",
				"an out-of-range rate must name its position too")
			assert.Contains(t, err.Error(), "quality_rate",
				"the refusal must name the rule it applied, so a stale rule reads as ours")
		})
	}
}

// TestValidateFactEntriesCollectsAll pins D-28-06: EVERY failure in the file is
// reported together, so fixing a two-hundred-entry file is one round trip
// rather than two hundred.
//
// The fixture carries two DIFFERENT failures at two DIFFERENT indices on
// purpose. A first-failure return satisfies SC2 just as well — no request is
// issued either way — so only an assertion naming both indices can tell a
// collect-all loop from an early return.
func TestValidateFactEntriesCollectsAll(t *testing.T) {
	entries := decodeFactEntries(t, `[
	  {"key":"documents_reviewed","value":10,"provenance":"GUESSED"},
	  {"key":"documents_reviewed","value":11,"provenance":"MEASURED"},
	  {"key":"quality_rate","value":1.5,"provenance":"DERIVED"}
	]`)

	err := validateFactEntries(entries)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry [0].provenance",
		"the first failure must be reported")
	assert.Contains(t, err.Error(), "entry [2].value",
		"the LAST failure must be reported too — this is what an early return loses")
	assert.NotContains(t, err.Error(), "entry [1]",
		"the valid entry must not be named")
}

// TestValidateFactEntriesProvenanceOptional pins Pitfall 6 and D-27-11.
//
// The schema documents provenance as "Optional. Defaults to SELF_REPORTED when
// omitted", so an omitted or explicitly null provenance is LEGAL and must not
// be refused. This is what makes reusing validateEnumField load-bearing rather
// than a convenience: its absent-or-null early return is exactly the rule a
// hand-rolled membership loop over the raw value would get wrong, refusing
// every entry that omits the field.
func TestValidateFactEntriesProvenanceOptional(t *testing.T) {
	t.Run("omitted entirely", func(t *testing.T) {
		entries := decodeFactEntries(t, `[
		  {"key":"documents_reviewed","value":10},
		  {"key":"documents_reviewed","value":11}
		]`)
		for _, entry := range entries {
			require.NotContains(t, entry, "provenance",
				"the fixture is only meaningful if NO entry carries the field")
		}
		assert.NoError(t, validateFactEntries(entries))
	})

	t.Run("explicitly null", func(t *testing.T) {
		entries := decodeFactEntries(t,
			`[{"key":"documents_reviewed","value":10,"provenance":null}]`)
		require.Contains(t, entries[0], "provenance",
			"an explicit null is PRESENT and nil, which is a different case from absent")
		assert.NoError(t, validateFactEntries(entries),
			"the field is nullable; null is not a value outside the closed set")
	})
}

// TestValidateFactEntriesSkipsShapeErrors pins that SHAPE is the API's
// authority, never this validator's (economics_doc.go:219-225).
//
// Note where the boundary actually falls. The behaviour was specified as "an
// array element that is a JSON string rather than an object produces no
// client-side error", but such an element can never reach validateFactEntries
// at all: the decode target is []map[string]interface{}, so readFactEntries
// refuses it one stage earlier. That earlier refusal is asserted here too, so
// the boundary is captured rather than assumed to be somewhere it is not.
//
// What the validator does see, and must let through untouched, is a
// well-formed entry whose FIELD types are wrong for the rule — a non-numeric
// value under the bounded key, or a non-string key. Refusing those would be
// this CLI inventing a shape rule the schema does not state.
func TestValidateFactEntriesSkipsShapeErrors(t *testing.T) {
	t.Run("a non-numeric value under the bounded key travels on", func(t *testing.T) {
		entries := decodeFactEntries(t, `[{"key":"quality_rate","value":"very good"}]`)
		assert.NoError(t, validateFactEntries(entries),
			"the type of value is the API's authority, not the rate bound's")
	})

	t.Run("a non-string key travels on", func(t *testing.T) {
		entries := decodeFactEntries(t, `[{"key":7,"value":1.5}]`)
		assert.NoError(t, validateFactEntries(entries))
	})

	t.Run("a non-object element is refused one stage earlier, by the reader", func(t *testing.T) {
		path := writeTempFactsFile(t, `["not an object"]`)
		_, err := readFactEntries(path, nil)
		require.Error(t, err,
			"the decode target is an array of objects, so this never reaches the validator")
		assert.Contains(t, err.Error(), path, "and the refusal names the input source")
	})
}

// TestReadFactEntriesRejects pins the four inputs that are NOT a usable array
// of entries, plus the array that is well-formed and empty.
//
// Each case asserts the SOURCE appears in the message: the path for a file, the
// word stdin for a pipe. "failed to parse" alone is nearly useless when a shell
// pipeline supplied the document and the operator has three files open — which
// is the property the analog's own doc comment says its tests exist to capture.
//
// The pipe cases are driven with a strings.Reader, never with the process's
// standard input: injecting the reader is the whole reason readFactEntries
// takes one.
func TestReadFactEntriesRejects(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		viaStdin bool
		contains []string
	}{
		{
			name:     "an empty file",
			body:     "",
			contains: []string{"is empty", "JSON array"},
		},
		{
			// A truncated pipe is the plausible cause, and the decoder's own
			// message for this input ("unexpected end of JSON input") does not
			// say what was wrong.
			name:     "a whitespace-only pipe",
			body:     "   \n\t  ",
			viaStdin: true,
			contains: []string{"stdin", "is empty", "JSON array"},
		},
		{
			// A bare null unmarshals into a NIL slice without an error, so
			// without an explicit branch it would travel on and be posted as
			// a JSON null rather than as an array.
			name:     "a bare null",
			body:     "null",
			viaStdin: true,
			contains: []string{"stdin", "null"},
		},
		{
			name:     "a JSON object rather than an array",
			body:     `{"key":"documents_reviewed","value":1}`,
			contains: []string{"JSON array"},
		},
		{
			// Well-formed and empty. An operator who piped a truncated file
			// must not read a success sentence saying nothing was appended.
			name:     "an empty array",
			body:     "[]",
			contains: []string{"nothing to append"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "-"
			var stdin *strings.Reader
			if tc.viaStdin {
				stdin = strings.NewReader(tc.body)
			} else {
				path = writeTempFactsFile(t, tc.body)
			}

			entries, err := readFactEntries(path, stdin)
			require.Error(t, err)
			assert.Nil(t, entries, "nothing may travel on from a refused input")
			for _, want := range tc.contains {
				assert.Contains(t, err.Error(), want)
			}
			if !tc.viaStdin {
				assert.Contains(t, err.Error(), path,
					"a file case must name the path the operator supplied")
			}
		})
	}
}
