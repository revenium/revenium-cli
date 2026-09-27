package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// This file holds the pure stages of the `jobs types facts append` pipeline —
// read and validate. Neither touches HTTP or cobra, and the read touches the
// filesystem through a single ReadFile, so both are decidable without a server
// and are pinned exhaustively in facts_doc_test.go. The command that composes
// them lives in facts_append.go:
//
//	readFactEntries -> validateFactEntries
//
// The entries travel as []map[string]interface{} end to end and are NEVER
// decoded into a typed struct. PeriodFactEntry declares no `required` key and
// the endpoint is append-only, so re-encoding through a struct would silently
// drop any property the struct does not know about on the day the spec grows
// one — and an append cannot be withdrawn and re-sent.

// readFactEntries reads a JSON array of PeriodFactEntry documents from a file
// path, or from the supplied reader when path is "-".
//
// It adopts the shape of readEconomicsInput, including that function's
// deliberate deviation from readBulkPricingInput: the reader is a parameter
// rather than a direct reference to the process's standard input, so the "-"
// path is testable without process-level plumbing. The decode target is an
// array because the endpoint's request body is an array of entries.
//
// Every error names the input source — the file path, or the word stdin.
// "failed to parse" without a source is nearly useless when a shell pipeline
// supplied the document.
func readFactEntries(path string, stdin io.Reader) ([]map[string]interface{}, error) {
	// economicsInputSource is reused verbatim rather than re-spelled: a second
	// `if path == "-"` in this package is one more place for two spellings of
	// the same rule to drift apart.
	source := economicsInputSource(path)

	var data []byte
	var err error
	if path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("failed to read fact entries from %s: no input reader was supplied", source)
		}
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read fact entries from %s: %w", source, err)
	}

	// An empty or whitespace-only document decodes to a json.Unmarshal
	// "unexpected end of JSON input", which does not say what was wrong. Name
	// the emptiness explicitly, before the decode is attempted — a truncated
	// pipe is a plausible cause.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("failed to parse fact entries from %s: input is empty, expected a JSON array", source)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse fact entries from %s as a JSON array: %w", source, err)
	}

	// A bare JSON `null` unmarshals into a nil slice WITHOUT an error, so this
	// branch is load-bearing rather than defensive: without it the nil slice
	// travels on and is posted as a JSON null where the endpoint's required
	// request body is an array.
	if entries == nil {
		return nil, fmt.Errorf("failed to parse fact entries from %s: expected a JSON array, got null", source)
	}

	// A well-formed empty array is refused rather than posted as a no-op. The
	// operator most likely piped a file that was truncated to its brackets,
	// and reading "Appended 0 period facts" as a success sentence would tell
	// them the month's measurements are recorded when nothing was sent.
	if len(entries) == 0 {
		return nil, fmt.Errorf("failed to read fact entries from %s: the array is empty, there is nothing to append", source)
	}

	return entries, nil
}

// factProvenanceValues is the closed FOUR-value set declared by the dev spec
// at PeriodFactEntry.properties.provenance.enum and, byte-identically, at
// OutcomeMetricEntry.properties.provenance.enum.
//
// It is deliberately NOT shared with baselineProvenanceValues in
// baselines_append.go, which is a different THREE-value set
// (CUSTOMER_DECLARED, MEASURED, SIGNED_OFF). MEASURED is the only member the
// two sets share, so a validator shared across them would accept SIGNED_OFF on
// a period fact and SELF_REPORTED on a baseline — each a word that means
// nothing to its own endpoint. The variable is named for its schema so that
// distinction survives a future reader who sees two provenance enums and
// assumes one. The analog already records the same warning from the other
// side; this closes the loop it opened.
//
// This check is also what goes stale if the server ever widens the set. That
// is intended: a rejection here should read as our check being out of date,
// not as a mystery, which is why the error names the values it accepts.
var factProvenanceValues = []string{"MEASURED", "SELF_REPORTED", "DERIVED", "ATTESTED"}

// qualityRateKey is the ONE metric key the spec bounds. The bound is stated
// exactly once, in the `value` property description on PeriodFactEntry,
// OutcomeMetricEntry and OutcomeMetricEntry_Read:
//
//	"Metric value. quality_rate is a rate and must be between 0 and 1."
//
// Rate-ness is keyed on this LITERAL key and is deliberately not derived from
// the metric type enum in JobTypeMetricDefinition: that enum has no rate
// member at all, and its nearest neighbour is conventionally a 0-to-100
// quantity, so inferring the bound from it would invent a rule and refuse
// input the API accepts (D-28-05, extending D-27-11). If the server ever
// bounds a second key, THIS constant is what goes stale, and the refusal names
// the rule so the staleness reads as ours.
const qualityRateKey = "quality_rate"

// validateFactEntries checks every entry and reports EVERY failure together,
// before any request is issued (JOBS-15 SC2, D-28-06).
//
// Collecting rather than returning on the first bad index: a first-failure
// return satisfies SC2 just as well, since no request is issued either way,
// but it makes fixing a two-hundred-entry file an N-round-trip chore against
// an endpoint whose every accepted entry is permanent.
//
// Two things it deliberately does NOT check. Required-ness of periodStart,
// periodEnd, dimensionKey, dimensionValue, key and value: PeriodFactEntry
// declares no `required` key at all, so a client-side guard would refuse input
// the API accepts (D-27-11, D-28-07). And SHAPE: an entry whose field types
// are wrong for a rule is skipped rather than refused, because shape is the
// API's authority (economics_doc.go:219-225).
func validateFactEntries(entries []map[string]interface{}) error {
	var problems []string
	invalid := 0

	for i, entry := range entries {
		path := fmt.Sprintf("entry [%d]", i)
		before := len(problems)

		// validateEnumField is reused unchanged, and the reuse is
		// load-bearing rather than a convenience: its absent-or-null early
		// return is exactly what a nullable field with a documented
		// server-side default needs. provenance is described "Optional.
		// Defaults to SELF_REPORTED when omitted", so a hand-rolled
		// membership loop over the raw value would refuse every entry that
		// omits the field. Its message already carries the path and the full
		// legal set, so passing an indexed path satisfies SC2's index
		// requirement with no new formatting code.
		if err := validateEnumField(entry, "provenance", path+".provenance", factProvenanceValues); err != nil {
			problems = append(problems, err.Error())
		}

		// The bound applies only under the literal bounded key. A JSON number
		// decodes into interface{} as a float64, so a value of any other type
		// is a shape error and travels to the API untouched.
		if key, ok := entry["key"].(string); ok && key == qualityRateKey {
			if v, ok := entry["value"].(float64); ok && (v < 0 || v > 1) {
				// Inclusive at both ends: the description states no
				// exclusivity, and refusing a legitimately perfect rate of
				// exactly 1 would be the refuse-what-the-API-accepts failure.
				problems = append(problems, fmt.Sprintf(
					"%s.value is %v, but %q is a rate and must be between 0 and 1 inclusive",
					path, v, qualityRateKey))
			}
		}

		if len(problems) > before {
			invalid++
		}
	}

	if len(problems) == 0 {
		return nil
	}

	// The count is of invalid ENTRIES, not of problems: one entry can fail two
	// rules, and "2 entries invalid" for a single bad entry would be a lie in
	// the one sentence the operator reads.
	return fmt.Errorf("%s invalid, nothing was appended:\n  %s",
		pluralEntries(invalid), strings.Join(problems, "\n  "))
}

// pluralSuffix renders the plural suffix for a count of nouns, so a success
// line can read "1 period fact" and "2 period facts" without a second noun
// helper. Refusal messages use pluralEntries (economics_doc.go), which renders
// the count and the noun together; this exists only for the success line.
func pluralSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
