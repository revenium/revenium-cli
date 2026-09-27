package jobs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// economicsRequestProperties are the SEVEN JobTypeEconomicsRequest properties.
//
// Seven, not six: overheadCurrency appears in neither the ROADMAP success
// criteria nor D-27-01's field list (correction C1), so a strip implemented
// from that wording would drop it. It is asserted by name below precisely
// because it is the one a reasonable reading of the requirement omits.
var economicsRequestProperties = []string{
	"unitMetricKey",
	"unitLabel",
	"metrics",
	"dimensions",
	"monetization",
	"overheadPerUnit",
	"overheadCurrency",
}

// writeTempEconomicsFile writes body to a file under t's temp dir and returns
// the path. The path is what the error messages must name, so tests capture it.
func writeTempEconomicsFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "economics.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

// decodeEconomicsFixture decodes a JSON body into the map shape the pipeline
// works in. Tests never decode into a struct — see the doc comment on
// readEconomicsInput for why the document stays a map end to end.
func decodeEconomicsFixture(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(body), &doc))
	return doc
}

// TestReadEconomicsInputFile: a path holding a JSON object decodes to a map
// carrying the same keys, including the two read-only ones — the strip stage,
// not the read stage, is what removes those.
func TestReadEconomicsInputFile(t *testing.T) {
	path := writeTempEconomicsFile(t, economicsResourceBody)

	doc, err := readEconomicsInput(path, nil)
	require.NoError(t, err)
	require.NotNil(t, doc)

	for _, key := range economicsRequestProperties {
		assert.Contains(t, doc, key)
	}
	assert.Contains(t, doc, "jobType")
	assert.Contains(t, doc, "currentBaseline")
	assert.Equal(t, "documents-reviewed", doc["unitMetricKey"])
}

// TestReadEconomicsInputStdin: with the path "-" the object is read from the
// supplied reader rather than from the process's standard input, which is the
// one deliberate deviation from readBulkPricingInput's shape and the reason
// this path is testable at all.
func TestReadEconomicsInputStdin(t *testing.T) {
	doc, err := readEconomicsInput("-", strings.NewReader(`{"unitMetricKey":"documents-reviewed","unitLabel":"documents reviewed"}`))
	require.NoError(t, err)

	assert.Equal(t, map[string]interface{}{
		"unitMetricKey": "documents-reviewed",
		"unitLabel":     "documents reviewed",
	}, doc)
}

// TestReadEconomicsInputRejectsNonObject: every input that is not a single
// JSON object is refused, and every refusal names the input source.
//
// "failed to parse" without a source is nearly useless when a shell pipeline
// supplied the document, so each case asserts the source appears in the
// message — the file path for a path, the word stdin for "-".
func TestReadEconomicsInputRejectsNonObject(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"array", `[{"unitMetricKey":"documents-reviewed"}]`},
		{"null", `null`},
		{"scalar", `"documents-reviewed"`},
		{"zero-byte", ``},
		{"whitespace-only", "  \n\t  "},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/file", func(t *testing.T) {
			path := writeTempEconomicsFile(t, tc.body)
			_, err := readEconomicsInput(path, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), path,
				"the error must name the input source — here, the file path")
		})

		t.Run(tc.name+"/stdin", func(t *testing.T) {
			_, err := readEconomicsInput("-", strings.NewReader(tc.body))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "stdin",
				"the error must name the input source — here, stdin")
		})
	}
}

// TestStripReadOnlyKeysResourceShape is the get/edit/set loop's first try:
// `economics get --json > f.json`, edit, `set --file f.json` must work without
// the operator hand-stripping keys the tool itself emitted.
//
// It asserts POSITIVELY that all seven request properties survive, naming
// overheadCurrency (correction C1), and that the caller's map is not mutated.
func TestStripReadOnlyKeysResourceShape(t *testing.T) {
	doc := decodeEconomicsFixture(t, economicsResourceBody)

	stripped, dropped := stripReadOnlyKeys(doc)

	assert.Equal(t, []string{"currentBaseline", "jobType"}, dropped,
		"both read-only keys are reported, sorted, so 27-05 can name them on stderr")
	assert.NotContains(t, stripped, "jobType")
	assert.NotContains(t, stripped, "currentBaseline")

	for _, key := range economicsRequestProperties {
		assert.Contains(t, stripped, key,
			"%s is a JobTypeEconomicsRequest property and must survive the strip", key)
	}
	assert.Len(t, stripped, len(economicsRequestProperties),
		"exactly the seven request properties remain")

	assert.Contains(t, doc, "jobType",
		"the caller's map must not be mutated — the strip returns a new map")
	assert.Contains(t, doc, "currentBaseline",
		"the caller's map must not be mutated — the strip returns a new map")
}

// TestStripReadOnlyKeysRequestShape: a document already in Request shape is
// returned unchanged with nothing dropped, so an operator who hand-authored a
// Request-shaped file sees no spurious note about keys they never wrote.
func TestStripReadOnlyKeysRequestShape(t *testing.T) {
	doc := decodeEconomicsFixture(t, `{
	  "unitMetricKey": "documents-reviewed",
	  "unitLabel": "documents reviewed",
	  "metrics": [],
	  "dimensions": [],
	  "monetization": null,
	  "overheadPerUnit": 12.5,
	  "overheadCurrency": "USD"
	}`)

	stripped, dropped := stripReadOnlyKeys(doc)

	assert.Empty(t, dropped)
	assert.Equal(t, doc, stripped)
}

// TestStripReadOnlyKeysPassesUnknownKeys: D-27-03 specifies stripping exactly
// two read-only keys and is silent on the rest, so a key in neither schema is
// passed through to the API rather than removed or reported as dropped.
// Rejecting or stripping unknowns would break the round-trip on the day the
// API grows a property; the API's 400/422 is the authority on everything else.
func TestStripReadOnlyKeysPassesUnknownKeys(t *testing.T) {
	doc := decodeEconomicsFixture(t, `{
	  "unitMetricKey": "documents-reviewed",
	  "somethingTheSpecGrewLastTuesday": {"nested": true},
	  "jobType": "acme-review"
	}`)

	stripped, dropped := stripReadOnlyKeys(doc)

	assert.Equal(t, []string{"jobType"}, dropped,
		"only the read-only key is dropped; the unknown key is not reported")
	assert.Contains(t, stripped, "somethingTheSpecGrewLastTuesday",
		"unknown keys are passed through untouched for forward compatibility")
	assert.Equal(t, map[string]interface{}{"nested": true}, stripped["somethingTheSpecGrewLastTuesday"])
}

// --- validateEconomicsDocument: the six closed enums -------------------------

// validMetricEntry returns a metrics entry whose four enum fields all carry
// legal values. Negative cases start from this and spoil exactly one field, so
// a failure can only be attributed to the field under test.
func validMetricEntry(key string) map[string]interface{} {
	return map[string]interface{}{
		"key":         key,
		"type":        "COUNT",
		"direction":   "HIGHER_IS_BETTER",
		"aggregation": "SUM",
		"resolution":  "PER_JOB",
	}
}

// validMonetization returns a JobTypeMonetization whose two enum fields carry
// legal values.
func validMonetization() map[string]interface{} {
	return map[string]interface{}{
		"metricKey":    "documents-reviewed",
		"valuePerUnit": 12.5,
		"currency":     "USD",
		"category":     "COST_AVOIDED",
		"basis":        "REALIZED",
	}
}

// economicsDocWithMetrics builds a document carrying entries as its metrics
// array, in the []interface{}-of-map[string]interface{} shape encoding/json
// actually produces — not []map[string]interface{}, which the decoder never
// yields and which a type assertion would silently miss.
func economicsDocWithMetrics(entries ...map[string]interface{}) map[string]interface{} {
	raw := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		raw = append(raw, entry)
	}
	return map[string]interface{}{
		"unitMetricKey": "documents-reviewed",
		"metrics":       raw,
	}
}

// requireEnumViolation asserts the document is refused with a message naming
// both the offending path — including its array index — and every legal value
// for that field.
//
// legal is passed as literals rather than as the production slice on purpose:
// an assertion that read the slice under test would move with it, and these
// tests exist to pin the enum against exactly that kind of drift.
func requireEnumViolation(t *testing.T, doc map[string]interface{}, path string, legal []string) {
	t.Helper()

	err := validateEconomicsDocument(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path,
		"inside a hand-edited JSON file, the message must name the offending path with its array index")
	for _, value := range legal {
		assert.Contains(t, err.Error(), value,
			"the message must list every legal value for %s — the API's own 422 is a generic \"Semantic validation failed\"", path)
	}
}

// The six negative cases below are deliberately six separate tests rather than
// one shared case. One shared case leaves five enums unpinned, which is exactly
// how a control ships green under the mutation it exists to catch.
//
// Each is pinned by widening its own list by the very value that case uses.

func TestValidateEconomicsDocumentRejectsMetricType(t *testing.T) {
	entry := validMetricEntry("documents-reviewed")
	entry["type"] = "TALLY"

	requireEnumViolation(t, economicsDocWithMetrics(entry), "metrics[0].type",
		[]string{"COUNT", "DURATION", "PERCENT", "MONEY", "SCORE"})
}

func TestValidateEconomicsDocumentRejectsMetricDirection(t *testing.T) {
	entry := validMetricEntry("minutes-per-document")
	entry["direction"] = "BIGGER_IS_BETTER"

	// Index 1: the message must carry the position, so the first entry is left
	// valid and the violation is placed second.
	doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"), entry)

	requireEnumViolation(t, doc, "metrics[1].direction",
		[]string{"HIGHER_IS_BETTER", "LOWER_IS_BETTER"})
}

func TestValidateEconomicsDocumentRejectsMetricAggregation(t *testing.T) {
	entry := validMetricEntry("quality-rate")
	entry["aggregation"] = "MEDIAN"

	doc := economicsDocWithMetrics(
		validMetricEntry("documents-reviewed"),
		validMetricEntry("minutes-per-document"),
		entry,
	)

	requireEnumViolation(t, doc, "metrics[2].aggregation",
		[]string{"SUM", "AVG", "LAST"})
}

func TestValidateEconomicsDocumentRejectsMetricResolution(t *testing.T) {
	entry := validMetricEntry("documents-reviewed")
	entry["resolution"] = "PER_PERIOD"

	requireEnumViolation(t, economicsDocWithMetrics(entry), "metrics[0].resolution",
		[]string{"PER_JOB", "PERIOD"})
}

// Correction C2: D-27-11 named only the four metric enums. This case and the
// next are the two it missed.
func TestValidateEconomicsDocumentRejectsMonetizationCategory(t *testing.T) {
	doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
	monetization := validMonetization()
	monetization["category"] = "GOODWILL"
	doc["monetization"] = monetization

	requireEnumViolation(t, doc, "monetization.category",
		[]string{"REVENUE", "COST_AVOIDED", "TIME_SAVED", "LEADING_VALUE"})
}

func TestValidateEconomicsDocumentRejectsMonetizationBasis(t *testing.T) {
	doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
	monetization := validMonetization()
	monetization["basis"] = "PROJECTED"
	doc["monetization"] = monetization

	requireEnumViolation(t, doc, "monetization.basis",
		[]string{"REALIZED", "EXPECTED"})
}

// TestValidateEconomicsDocumentAcceptsValid exercises every legal value of all
// six enums.
func TestValidateEconomicsDocumentAcceptsValid(t *testing.T) {
	types := []string{"COUNT", "DURATION", "PERCENT", "MONEY", "SCORE"}
	directions := []string{"HIGHER_IS_BETTER", "LOWER_IS_BETTER"}
	aggregations := []string{"SUM", "AVG", "LAST"}
	resolutions := []string{"PER_JOB", "PERIOD"}
	categories := []string{"REVENUE", "COST_AVOIDED", "TIME_SAVED", "LEADING_VALUE"}
	bases := []string{"REALIZED", "EXPECTED"}

	// Every combination of the four metric enums in one document: 5*2*3*2 = 60
	// entries, so every legal value of all four appears at least once.
	var entries []map[string]interface{}
	for _, metricType := range types {
		for _, direction := range directions {
			for _, aggregation := range aggregations {
				for _, resolution := range resolutions {
					entries = append(entries, map[string]interface{}{
						"key":         fmt.Sprintf("%s-%s-%s-%s", metricType, direction, aggregation, resolution),
						"type":        metricType,
						"direction":   direction,
						"aggregation": aggregation,
						"resolution":  resolution,
					})
				}
			}
		}
	}
	require.NoError(t, validateEconomicsDocument(economicsDocWithMetrics(entries...)))

	// monetization is a single object, so its two enums are exercised across
	// one document per combination.
	for _, category := range categories {
		for _, basis := range bases {
			doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
			monetization := validMonetization()
			monetization["category"] = category
			monetization["basis"] = basis
			doc["monetization"] = monetization

			assert.NoError(t, validateEconomicsDocument(doc),
				"category=%s basis=%s are both legal and must be accepted", category, basis)
		}
	}

	// A real server document, decoded from JSON rather than hand-built, also
	// validates clean — the round trip must not be refused by our own check.
	require.NoError(t, validateEconomicsDocument(decodeEconomicsFixture(t, economicsResourceBody)))
}

// TestValidateEconomicsDocumentEnumsAreCaseSensitive proves membership is exact
// byte equality rather than a case-insensitive match: `sum` is refused where
// `SUM` is accepted. A ToUpper or EqualFold "convenience" here would accept a
// value the API refuses, turning a local error into a round trip.
func TestValidateEconomicsDocumentEnumsAreCaseSensitive(t *testing.T) {
	lower := validMetricEntry("documents-reviewed")
	lower["aggregation"] = "sum"
	require.Error(t, validateEconomicsDocument(economicsDocWithMetrics(lower)),
		"`sum` is not `SUM` — enum membership is exact and case-sensitive")

	upper := validMetricEntry("documents-reviewed")
	upper["aggregation"] = "SUM"
	require.NoError(t, validateEconomicsDocument(economicsDocWithMetrics(upper)))
}

// TestValidateEconomicsDocumentSkipsAbsentFields: a closed set constrains the
// values a field may take, not whether the field appears. The schemas declare
// no `required` key at all, so required-ness goes to the API.
func TestValidateEconomicsDocumentSkipsAbsentFields(t *testing.T) {
	t.Run("metric entry omitting every enum field", func(t *testing.T) {
		doc := economicsDocWithMetrics(map[string]interface{}{"key": "documents-reviewed"})
		require.NoError(t, validateEconomicsDocument(doc))
	})

	t.Run("metric entry with a null enum field", func(t *testing.T) {
		entry := validMetricEntry("documents-reviewed")
		entry["resolution"] = nil
		require.NoError(t, validateEconomicsDocument(economicsDocWithMetrics(entry)))
	})

	t.Run("no monetization at all", func(t *testing.T) {
		doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
		require.NoError(t, validateEconomicsDocument(doc))
	})

	t.Run("monetization explicitly null", func(t *testing.T) {
		doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
		doc["monetization"] = nil
		require.NoError(t, validateEconomicsDocument(doc))
	})

	t.Run("empty document", func(t *testing.T) {
		require.NoError(t, validateEconomicsDocument(map[string]interface{}{}))
	})
}

// TestValidateEconomicsDocumentIgnoresRanges proves no invented rule is
// enforced. Neither constraint below is declared in any schema: the 0-1 rate
// bound belongs to Phase 28's outcome-fact language and is absent from these
// schemas, and "Currently must be USD" is a schema *description* on a property
// typed string, not an enum. Enforcing either would refuse input the API
// accepts. Ranges, required-ness and URL shape go to the API.
func TestValidateEconomicsDocumentIgnoresRanges(t *testing.T) {
	doc := economicsDocWithMetrics(validMetricEntry("documents-reviewed"))
	monetization := validMonetization()
	monetization["currency"] = "EUR"
	monetization["valuePerUnit"] = -400.0
	doc["monetization"] = monetization
	doc["overheadCurrency"] = "GBP"
	doc["overheadPerUnit"] = -17.5
	doc["someRate"] = 4.2 // far outside any 0-1 bound

	require.NoError(t, validateEconomicsDocument(doc),
		"no range rule and no USD rule may be invented — neither is declared in any schema")
}

// --- diffEconomics: what a replace would destroy -----------------------------

// dimensionEntry builds a JobTypeDimensionDefinition in the shape
// encoding/json produces: allowedValues is []interface{} of string, not
// []string.
func dimensionEntry(key string, allowedValues ...string) map[string]interface{} {
	raw := make([]interface{}, 0, len(allowedValues))
	for _, value := range allowedValues {
		raw = append(raw, value)
	}
	return map[string]interface{}{"key": key, "allowedValues": raw}
}

// currentContract is the "current" side of every diff below: two metrics, one
// dimension carrying two allowed values, a monetization object, and a
// unitMetricKey. Two metrics rather than one so a test can drop exactly one
// and still prove the other survived.
func currentContract() map[string]interface{} {
	return map[string]interface{}{
		"unitMetricKey":    "documents-reviewed",
		"unitLabel":        "documents reviewed",
		"overheadPerUnit":  1234567.0,
		"overheadCurrency": "USD",
		"metrics": []interface{}{
			validMetricEntry("documents-reviewed"),
			validMetricEntry("minutes-per-document"),
		},
		"dimensions": []interface{}{
			dimensionEntry("region", "us-east", "eu-west"),
		},
		"monetization": validMonetization(),
	}
}

// --- destructive classes (five) ---

// A metrics entry present in the current contract is absent from the next
// document. Explicit in D-27-04.
func TestDiffEconomicsDroppedMetricIsDestructive(t *testing.T) {
	next := currentContract()
	next["metrics"] = []interface{}{validMetricEntry("documents-reviewed")}

	diff := diffEconomics(currentContract(), next)

	assert.True(t, diff.IsDestructive())
	assert.Equal(t, []string{"minutes-per-document"}, diff.LostMetricKeys)
	assert.Empty(t, diff.LostDimensionKeys)
	assert.False(t, diff.MonetizationRemoved)
}

// A dimensions entry present in the current contract is absent. Explicit in
// D-27-04.
func TestDiffEconomicsDroppedDimensionIsDestructive(t *testing.T) {
	next := currentContract()
	next["dimensions"] = []interface{}{}

	diff := diffEconomics(currentContract(), next)

	assert.True(t, diff.IsDestructive())
	assert.Equal(t, []string{"region"}, diff.LostDimensionKeys)
	assert.Empty(t, diff.LostMetricKeys)
}

// An allowedValues entry is dropped from a dimension that is itself retained:
// the same class of loss one level down, since a declared value silently stops
// being accepted.
func TestDiffEconomicsDroppedAllowedValueIsDestructive(t *testing.T) {
	next := currentContract()
	next["dimensions"] = []interface{}{dimensionEntry("region", "us-east")}

	diff := diffEconomics(currentContract(), next)

	assert.True(t, diff.IsDestructive())
	assert.Empty(t, diff.LostDimensionKeys, "the dimension itself is retained")
	assert.Equal(t, map[string][]string{"region": {"eu-west"}}, diff.LostAllowedValues)
}

// monetization goes from a non-null object to null or absent: this deletes the
// entire value-derivation rule and nothing else records it.
func TestDiffEconomicsMonetizationRemovalIsDestructive(t *testing.T) {
	t.Run("explicitly null", func(t *testing.T) {
		next := currentContract()
		next["monetization"] = nil

		diff := diffEconomics(currentContract(), next)

		assert.True(t, diff.IsDestructive())
		assert.True(t, diff.MonetizationRemoved)
	})

	t.Run("absent", func(t *testing.T) {
		next := currentContract()
		delete(next, "monetization")

		diff := diffEconomics(currentContract(), next)

		assert.True(t, diff.IsDestructive())
		assert.True(t, diff.MonetizationRemoved)
	})
}

// unitMetricKey changes to a different non-empty value. It is the anchor every
// baseline's per-unit figure is denominated in, so re-anchoring silently makes
// historical baselines mean something different.
func TestDiffEconomicsChangedUnitMetricKeyIsDestructive(t *testing.T) {
	next := currentContract()
	next["unitMetricKey"] = "minutes-per-document"

	diff := diffEconomics(currentContract(), next)

	assert.True(t, diff.IsDestructive())
	assert.True(t, diff.UnitMetricKeyChanged)
	assert.Equal(t, "documents-reviewed", diff.OldUnitMetricKey)
	assert.Equal(t, "minutes-per-document", diff.NewUnitMetricKey)
}

// A unitMetricKey that is dropped entirely is a strictly larger loss than one
// that is re-pointed, so it is classified the same way. Without this, a
// document that keeps every metric but omits the anchor would slip past the
// guard — the one hole the "changed value" rule alone leaves open.
func TestDiffEconomicsRemovedUnitMetricKeyIsDestructive(t *testing.T) {
	next := currentContract()
	delete(next, "unitMetricKey")

	diff := diffEconomics(currentContract(), next)

	assert.True(t, diff.IsDestructive())
	assert.True(t, diff.UnitMetricKeyChanged)
	assert.Equal(t, "documents-reviewed", diff.OldUnitMetricKey)
	assert.Empty(t, diff.NewUnitMetricKey)
}

// --- non-destructive classes (four) ---

// Additions are never destructive.
func TestDiffEconomicsAddedEntriesAreNotDestructive(t *testing.T) {
	next := currentContract()
	next["metrics"] = []interface{}{
		validMetricEntry("documents-reviewed"),
		validMetricEntry("minutes-per-document"),
		validMetricEntry("quality-rate"),
	}
	next["dimensions"] = []interface{}{
		dimensionEntry("region", "us-east", "eu-west", "ap-south"),
		dimensionEntry("team", "legal", "finance"),
	}

	diff := diffEconomics(currentContract(), next)

	assert.False(t, diff.IsDestructive())
	assert.Empty(t, diff.LostMetricKeys)
	assert.Empty(t, diff.LostDimensionKeys)
	assert.Empty(t, diff.LostAllowedValues)
}

// unitLabel, overheadPerUnit and overheadCurrency are scalar edits whose
// replaced value is visible in the round-trip.
func TestDiffEconomicsScalarEditsAreNotDestructive(t *testing.T) {
	next := currentContract()
	next["unitLabel"] = "documents processed"
	next["overheadPerUnit"] = 42.0
	next["overheadCurrency"] = "GBP"

	diff := diffEconomics(currentContract(), next)

	assert.False(t, diff.IsDestructive())
	assert.False(t, diff.UnitMetricKeyChanged)
}

// Identical documents lose nothing. A single-entry metrics array that drops
// nothing is likewise non-destructive.
func TestDiffEconomicsIdenticalDocumentsAreNotDestructive(t *testing.T) {
	diff := diffEconomics(currentContract(), currentContract())
	assert.False(t, diff.IsDestructive())

	single := map[string]interface{}{
		"unitMetricKey": "documents-reviewed",
		"metrics":       []interface{}{validMetricEntry("documents-reviewed")},
	}
	assert.False(t, diffEconomics(single, single).IsDestructive(),
		"a single-entry metrics array that drops nothing is not destructive")
}

// An empty or absent current contract means nothing exists to lose — this is
// the create case, which `set` must support (PUT is documented as an upsert).
func TestDiffEconomicsEmptyCurrentIsNotDestructive(t *testing.T) {
	diff := diffEconomics(map[string]interface{}{}, currentContract())

	assert.False(t, diff.IsDestructive())
	assert.Empty(t, diff.LostMetricKeys)
	assert.Empty(t, diff.LostDimensionKeys)
	assert.False(t, diff.MonetizationRemoved)
	assert.False(t, diff.UnitMetricKeyChanged,
		"setting the anchor for the first time loses nothing")
}

// --- summary, key identity, and the partial-file case ---

// The summary is what an operator reads in a refusal, so "3 items would be
// lost" is not sufficient: the keys are the actionable content, and each loss
// must say which collection it came from.
func TestDiffEconomicsSummaryNamesLosses(t *testing.T) {
	next := map[string]interface{}{
		"unitMetricKey": "documents-reviewed",
		"metrics":       []interface{}{validMetricEntry("documents-reviewed")},
		"dimensions":    []interface{}{dimensionEntry("region", "us-east")},
	}

	diff := diffEconomics(currentContract(), next)
	summary := diff.Summary()

	require.True(t, diff.IsDestructive())
	assert.Contains(t, summary, "minutes-per-document", "the lost metric must be named")
	assert.Contains(t, summary, "eu-west", "the lost allowed value must be named")
	assert.Contains(t, summary, "region", "the dimension the value was lost from must be named")
	assert.Contains(t, summary, "metric", "each loss must say which collection it came from")
	assert.Contains(t, summary, "dimension", "each loss must say which collection it came from")
	assert.Contains(t, summary, "monetization", "the removed monetization rule must be named")
}

// Entry identity is exact byte equality of the UTF-8 key string: no case
// folding, no trimming, no Unicode normalization. `Region` and `region` are two
// different dimensions. Stating it in a test prevents a later "helpful"
// normalization from silently merging two declared dimensions into one.
func TestDiffEconomicsKeyEqualityIsExact(t *testing.T) {
	current := map[string]interface{}{
		"dimensions": []interface{}{dimensionEntry("region", "us-east")},
	}
	next := map[string]interface{}{
		"dimensions": []interface{}{dimensionEntry("Region", "us-east")},
	}

	diff := diffEconomics(current, next)

	assert.True(t, diff.IsDestructive(),
		"`region` is lost and `Region` is an addition — they are not the same key")
	assert.Equal(t, []string{"region"}, diff.LostDimensionKeys)
}

// An empty JSON object is exactly the partial document a first-time operator
// hand-authors, and it is the case D-27-04 exists to catch: it parses, strips
// nothing, validates clean, and drops every metric and dimension that exists.
func TestDiffEconomicsEmptyObjectDropsEverything(t *testing.T) {
	next := map[string]interface{}{}

	// It reaches the diff unrefused by the earlier stages — that is what makes
	// the diff the last line of defence for this input.
	stripped, dropped := stripReadOnlyKeys(next)
	require.Empty(t, dropped)
	require.NoError(t, validateEconomicsDocument(stripped))

	diff := diffEconomics(currentContract(), stripped)

	assert.True(t, diff.IsDestructive())
	assert.Equal(t, []string{"documents-reviewed", "minutes-per-document"}, diff.LostMetricKeys)
	assert.Equal(t, []string{"region"}, diff.LostDimensionKeys)
	assert.True(t, diff.MonetizationRemoved)

	summary := diff.Summary()
	assert.Contains(t, summary, "documents-reviewed")
	assert.Contains(t, summary, "minutes-per-document")
	assert.Contains(t, summary, "region")
}

// --- WR-01: entries the diff cannot track ------------------------------------

// An entry carrying no string `key` is invisible to the index in BOTH
// documents, so dropping it used to register as no loss at all — and the diff
// then stated, positively and falsely, that "nothing declared in the current
// contract would be lost".
//
// This is reachable rather than theoretical: every schema in play declares
// `required: []` and validateEconomicsDocument deliberately checks only the six
// closed enums, so a key-less metric entry passes the client validator and the
// API. Once such a contract exists, the next `set` erases it silently.
//
// The fail-closed reading is the one this file adopts everywhere else: an
// untrackable entry is precisely the case where the tool CANNOT prove nothing
// is lost, so it must require --yes.
func TestDiffEconomicsUntrackableCurrentEntryIsDestructive(t *testing.T) {
	t.Run("a key-less metric in the current contract", func(t *testing.T) {
		current := map[string]interface{}{
			"unitMetricKey": "documents-reviewed",
			"metrics": []interface{}{
				map[string]interface{}{"type": "COUNT", "direction": "HIGHER_IS_BETTER"},
			},
		}
		next := map[string]interface{}{"unitMetricKey": "documents-reviewed"}

		diff := diffEconomics(current, next)

		assert.True(t, diff.IsDestructive(),
			"an entry that cannot be matched across documents cannot be proven safe")
		assert.Equal(t, map[string]int{"metrics": 1}, diff.UntrackableEntries)

		// Exact, distinctive text: asserting merely on "metrics" would be
		// satisfied by the lost-metrics clause, which this case does not and
		// cannot produce.
		assert.Contains(t, diff.Summary(), "cannot be proven safe")
		assert.Contains(t, diff.Summary(), `1 entry in the current contract's "metrics" declares no key`)
	})

	t.Run("a key-less dimension in the current contract", func(t *testing.T) {
		current := map[string]interface{}{
			"dimensions": []interface{}{
				map[string]interface{}{"allowedValues": []interface{}{"us-east", "eu-west"}},
			},
		}

		diff := diffEconomics(current, map[string]interface{}{})

		assert.True(t, diff.IsDestructive())
		assert.Equal(t, map[string]int{"dimensions": 1}, diff.UntrackableEntries)
		assert.Contains(t, diff.Summary(), `1 entry in the current contract's "dimensions" declares no key`)
	})

	t.Run("counts are per collection and the wording pluralises", func(t *testing.T) {
		current := map[string]interface{}{
			"metrics": []interface{}{
				map[string]interface{}{"type": "COUNT"},
				map[string]interface{}{"key": 7}, // a key that is not a string
				validMetricEntry("documents-reviewed"),
			},
			"dimensions": []interface{}{
				map[string]interface{}{"allowedValues": []interface{}{"us-east"}},
			},
		}

		diff := diffEconomics(current, currentContract())

		assert.Equal(t, map[string]int{"metrics": 2, "dimensions": 1}, diff.UntrackableEntries)
		assert.Contains(t, diff.Summary(), `2 entries in the current contract's "metrics" declare no key`)
		assert.Contains(t, diff.Summary(), `1 entry in the current contract's "dimensions" declares no key`)
	})

	t.Run("an untrackable entry in the NEXT document alone is not destructive", func(t *testing.T) {
		next := currentContract()
		next["metrics"] = append(next["metrics"].([]interface{}),
			map[string]interface{}{"type": "SCORE"})

		diff := diffEconomics(currentContract(), next)

		assert.False(t, diff.IsDestructive(),
			"nothing declared in the CURRENT contract is at risk — the guard is about loss, not tidiness")
		assert.Empty(t, diff.UntrackableEntries)
	})
}

// --- WR-02: duplicate entry keys ---------------------------------------------

// Two entries under one key is a plausible copy-paste result and nothing
// rejects it: the schemas declare no uniqueness constraint and
// validateEconomicsDocument does not look at keys at all, so a hand-edited file
// that duplicates a dimension round-trips into the server. The next `set` then
// lost everything the earlier duplicate declared, because the index kept only
// the last entry seen while recording the key once.
func TestDiffEconomicsDuplicateKeysDoNotHideLoss(t *testing.T) {
	t.Run("duplicates in the current contract are unioned", func(t *testing.T) {
		current := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "us-east"),
			dimensionEntry("region", "eu-west"),
		}}
		next := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "eu-west"),
		}}

		diff := diffEconomics(current, next)

		assert.True(t, diff.IsDestructive(),
			`"us-east" is declared by the current contract and is not in the supplied document`)
		assert.Equal(t, map[string][]string{"region": {"us-east"}}, diff.LostAllowedValues)
		assert.Contains(t, diff.Summary(), "us-east")
	})

	t.Run("the union follows document order across duplicates", func(t *testing.T) {
		current := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "us-east", "eu-west"),
			dimensionEntry("region", "ap-south"),
		}}
		next := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "eu-west"),
		}}

		diff := diffEconomics(current, next)

		assert.Equal(t, map[string][]string{"region": {"us-east", "ap-south"}}, diff.LostAllowedValues,
			"losses read in the order the operator's own contract declares them")
	})

	t.Run("duplicates in the supplied document credit only what every entry declares", func(t *testing.T) {
		current := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "us-east", "eu-west"),
		}}
		next := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "us-east"),
			dimensionEntry("region", "eu-west"),
		}}

		diff := diffEconomics(current, next)

		assert.True(t, diff.IsDestructive(),
			"we cannot know which duplicate the server honours, so neither value is proven to survive")
		assert.Equal(t, map[string][]string{"region": {"us-east", "eu-west"}}, diff.LostAllowedValues)
	})

	t.Run("a single entry on each side is unaffected", func(t *testing.T) {
		current := map[string]interface{}{"dimensions": []interface{}{
			dimensionEntry("region", "us-east", "eu-west"),
		}}

		assert.False(t, diffEconomics(current, current).IsDestructive(),
			"the duplicate handling must not make an ordinary contract look destructive")
	})
}

// --- WR-03: retained but redefined -------------------------------------------

// The diff compared entry KEYS only, so a metric that kept its key while
// changing what its number means was "retained" and lost nothing. That is the
// same fact pattern the file already classifies as destructive one level up: a
// re-pointed unitMetricKey is destructive because it "silently makes historical
// baselines mean something different", which is true verbatim of a retained
// metric's aggregation and resolution.
//
// The comparison rule is deliberately the same as unitMetricKey's: a field the
// current contract does not declare is empty-to-anything and loses nothing (the
// create case), while a declared value that is changed OR omitted is a loss.
func TestDiffEconomicsRedefinedMetricIsDestructive(t *testing.T) {
	redefine := func(field, value string) economicsDiff {
		entry := validMetricEntry("documents-reviewed")
		entry[field] = value
		next := currentContract()
		next["metrics"] = []interface{}{entry, validMetricEntry("minutes-per-document")}
		return diffEconomics(currentContract(), next)
	}

	for field, value := range map[string]string{
		"type":        "SCORE",
		"direction":   "LOWER_IS_BETTER",
		"aggregation": "LAST",
		"resolution":  "PERIOD",
	} {
		t.Run("changed "+field, func(t *testing.T) {
			diff := redefine(field, value)

			assert.True(t, diff.IsDestructive(),
				"%s defines what the metric's number means", field)
			assert.Equal(t, []string{"documents-reviewed"}, diff.RedefinedMetricKeys)
			assert.Empty(t, diff.LostMetricKeys, "the key itself is retained")
			assert.Contains(t, diff.Summary(), "would be redefined")
			assert.Contains(t, diff.Summary(), "documents-reviewed")
		})
	}

	t.Run("an omitted field is a redefinition too", func(t *testing.T) {
		entry := validMetricEntry("documents-reviewed")
		delete(entry, "aggregation")
		next := currentContract()
		next["metrics"] = []interface{}{entry, validMetricEntry("minutes-per-document")}

		diff := diffEconomics(currentContract(), next)

		assert.True(t, diff.IsDestructive(),
			"dropping a declared aggregation is a strictly larger loss than changing it")
		assert.Equal(t, []string{"documents-reviewed"}, diff.RedefinedMetricKeys)
	})

	t.Run("declaring a field the current contract did not is not destructive", func(t *testing.T) {
		bare := map[string]interface{}{"key": "documents-reviewed", "type": "COUNT"}
		current := map[string]interface{}{"metrics": []interface{}{bare}}
		next := map[string]interface{}{
			"metrics": []interface{}{validMetricEntry("documents-reviewed")},
		}

		diff := diffEconomics(current, next)

		assert.False(t, diff.IsDestructive(),
			"empty-to-anything is the create case, exactly as it is for unitMetricKey")
		assert.Empty(t, diff.RedefinedMetricKeys)
	})

	t.Run("an unchanged metric is not redefined", func(t *testing.T) {
		diff := diffEconomics(currentContract(), currentContract())
		assert.False(t, diff.IsDestructive())
		assert.Empty(t, diff.RedefinedMetricKeys)
	})
}

// A monetization object that keeps being an object while re-pointing metricKey
// at a different metric was judged safe on the very rationale that makes a
// re-pointed unitMetricKey destructive: metricKey selects which metric the
// value rule reads.
//
// valuePerUnit and currency are deliberately NOT part of this check. They are
// scalar edits whose replaced value is visible in the round-trip, which is the
// same reason unitLabel, overheadPerUnit and overheadCurrency stay
// non-destructive. metricKey, category and basis determine what the rule
// computes and how the resulting value is booked.
func TestDiffEconomicsRedefinedMonetizationIsDestructive(t *testing.T) {
	t.Run("the review's reproduction: every declared semantic replaced", func(t *testing.T) {
		current := map[string]interface{}{
			"metrics": []interface{}{map[string]interface{}{
				"key": "k", "type": "MONEY", "aggregation": "SUM", "resolution": "PER_JOB",
			}},
			"monetization": map[string]interface{}{
				"metricKey": "k", "valuePerUnit": 100.0,
				"category": "REVENUE", "basis": "REALIZED",
			},
		}
		next := map[string]interface{}{
			"metrics": []interface{}{map[string]interface{}{
				"key": "k", "type": "SCORE", "aggregation": "LAST", "resolution": "PERIOD",
			}},
			"monetization": map[string]interface{}{
				"metricKey": "other", "valuePerUnit": 0.0,
				"category": "TIME_SAVED", "basis": "EXPECTED",
			},
		}

		diff := diffEconomics(current, next)

		assert.True(t, diff.IsDestructive())
		assert.Equal(t, []string{"k"}, diff.RedefinedMetricKeys)
		assert.Equal(t, []string{"metricKey", "category", "basis"}, diff.RedefinedMonetizationFields,
			"reported in schema order, so the message is deterministic")
		assert.False(t, diff.MonetizationRemoved, "the object is retained, not removed")

		summary := diff.Summary()
		assert.Contains(t, summary, "the monetization rule would be redefined (metricKey, category, basis)")
	})

	t.Run("re-pointing metricKey alone", func(t *testing.T) {
		next := currentContract()
		monetization := validMonetization()
		monetization["metricKey"] = "minutes-per-document"
		next["monetization"] = monetization

		diff := diffEconomics(currentContract(), next)

		assert.True(t, diff.IsDestructive(),
			"metricKey selects which metric the value rule reads — the unitMetricKey argument verbatim")
		assert.Equal(t, []string{"metricKey"}, diff.RedefinedMonetizationFields)
	})

	t.Run("valuePerUnit and currency are scalar edits, not redefinitions", func(t *testing.T) {
		next := currentContract()
		monetization := validMonetization()
		monetization["valuePerUnit"] = 0.0
		monetization["currency"] = "EUR"
		next["monetization"] = monetization

		diff := diffEconomics(currentContract(), next)

		assert.False(t, diff.IsDestructive(),
			"their replaced value is visible in the round-trip, like overheadPerUnit")
		assert.Empty(t, diff.RedefinedMonetizationFields)
	})

	t.Run("a removed monetization is reported as removed, not redefined", func(t *testing.T) {
		next := currentContract()
		next["monetization"] = nil

		diff := diffEconomics(currentContract(), next)

		assert.True(t, diff.MonetizationRemoved)
		assert.Empty(t, diff.RedefinedMonetizationFields,
			"a rule that is gone is not a rule that changed — reporting both would say it twice")
	})
}
