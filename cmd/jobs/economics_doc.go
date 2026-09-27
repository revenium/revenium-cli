package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// This file holds the four pure stages of the `economics set` document
// pipeline — read, strip, validate, diff. None of them touches HTTP, cobra or
// the filesystem beyond a single ReadFile, so each is decidable without a
// server and pinned exhaustively in economics_doc_test.go. The command that
// composes them lives in economics_set.go:
//
//	readEconomicsInput -> stripReadOnlyKeys -> validateEconomicsDocument -> diffEconomics
//
// The document is carried as a map[string]interface{} end to end and is NEVER
// decoded into a typed struct. Re-encoding a struct for the PUT would silently
// drop any server field the struct does not know about, breaking the
// get/edit/set round-trip (D-27-03) on the day the spec grows a property.

// readOnlyEconomicsKeys are exactly the JobTypeEconomicsResource properties
// absent from JobTypeEconomicsRequest, derived from Resource-minus-Request in
// the dev spec (components.schemas). If the API adds a read-only key, THIS
// LIST is what goes stale, and tools/coverage-audit is what reports it — the
// Go suite deliberately does not read the cached OpenAPI document (D-27-12).
var readOnlyEconomicsKeys = []string{"jobType", "currentBaseline"}

// economicsInputSource names the input for an error message: the file path, or
// the word stdin when the path was "-". "failed to parse" without a source is
// nearly useless when a shell pipeline supplied the document.
func economicsInputSource(path string) string {
	if path == "-" {
		return "stdin"
	}
	return path
}

// readEconomicsInput reads one JSON object from a file path, or from stdin
// when path is "-".
//
// It adopts the shape of readBulkPricingInput (cmd/models/pricing_bulk_save.go)
// per D-27-01, with two deliberate deviations:
//
//  1. The reader is a parameter rather than a direct reference to the
//     process's standard input, so the stdin path is testable without
//     process-level plumbing.
//  2. The decode target is a single object, not an array —
//     JobTypeEconomicsRequest is one document.
//
// Anything that is not a single JSON object is an error: an array, a bare
// null, a bare scalar, empty input and whitespace-only input alike. Every such
// error names the input source.
func readEconomicsInput(path string, stdin io.Reader) (map[string]interface{}, error) {
	source := economicsInputSource(path)

	var data []byte
	var err error
	if path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("failed to read economics document from %s: no input reader was supplied", source)
		}
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read economics document from %s: %w", source, err)
	}

	// An empty or whitespace-only document decodes to a json.Unmarshal
	// "unexpected end of JSON input", which does not say what was wrong. Name
	// the emptiness explicitly — a truncated pipe is a plausible cause.
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("failed to parse economics document from %s: input is empty, expected a JSON object", source)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse economics document from %s as a JSON object: %w", source, err)
	}

	// A bare JSON `null` unmarshals into a nil map WITHOUT an error, so this
	// branch is load-bearing rather than defensive: without it, `null` would
	// travel on as an empty document and diff as dropping every metric and
	// dimension that currently exists.
	if doc == nil {
		return nil, fmt.Errorf("failed to parse economics document from %s: expected a JSON object, got null", source)
	}

	return doc, nil
}

// stripReadOnlyKeys returns a copy of doc without the read-only keys, plus the
// sorted names of the keys actually dropped.
//
// This is what makes `economics get --json > f.json` -> edit -> `set --file
// f.json` work on the first try (D-27-03): the Resource shape the tool itself
// emits carries two keys the Request does not accept, and an operator should
// not have to hand-strip them. The dropped names are returned rather than
// swallowed so the caller can say WHICH keys it ignored — an operator who
// edited currentBaseline must not be left believing the baseline moved, since
// baselines are append-only.
//
// Every other key is copied through untouched, INCLUDING keys in neither
// schema. D-27-03 specifies stripping exactly two keys and is silent on the
// rest; passing unknowns through preserves forward compatibility with a spec
// that grows a property, and the API's 400/422 remains the authority on
// everything else.
//
// The caller's map is never mutated.
func stripReadOnlyKeys(doc map[string]interface{}) (map[string]interface{}, []string) {
	readOnly := make(map[string]bool, len(readOnlyEconomicsKeys))
	for _, key := range readOnlyEconomicsKeys {
		readOnly[key] = true
	}

	stripped := make(map[string]interface{}, len(doc))
	dropped := []string{}
	for key, value := range doc {
		if readOnly[key] {
			dropped = append(dropped, key)
			continue
		}
		stripped[key] = value
	}

	// Sorted so the operator-facing note is deterministic; Go map iteration
	// order is not.
	sort.Strings(dropped)
	return stripped, dropped
}

// The SIX closed enums this document carries. Each slice names its exact
// schema location in the dev spec, so a server-side widening reads as our
// check being stale rather than as a mystery (D-27-11's staleness guard).
//
// Six, not four. D-27-11 names only the four JobTypeMetricDefinition enums;
// JobTypeMonetization.category and .basis are equally closed and equally worth
// catching before a round trip, since the API's own rejection is a generic
// "Semantic validation failed" that need not name the legal values. That is
// correction C2, and a validation test file with exactly four negative cases
// is the warning sign that it was missed.
//
// Membership is exact, case-sensitive byte equality against these literals:
// `sum` is a rejection and `SUM` is an acceptance. No ToUpper, no EqualFold,
// no trimming. This is the same equality rule diffEconomics uses for entry
// keys, and stating it here prevents a later "helpful" normalization from
// quietly accepting a value the API will refuse.
var (
	// metricTypeValues: components.schemas.JobTypeMetricDefinition.properties.type.enum
	metricTypeValues = []string{"COUNT", "DURATION", "PERCENT", "MONEY", "SCORE"}
	// metricDirectionValues: components.schemas.JobTypeMetricDefinition.properties.direction.enum
	metricDirectionValues = []string{"HIGHER_IS_BETTER", "LOWER_IS_BETTER"}
	// metricAggregationValues: components.schemas.JobTypeMetricDefinition.properties.aggregation.enum
	metricAggregationValues = []string{"SUM", "AVG", "LAST"}
	// metricResolutionValues: components.schemas.JobTypeMetricDefinition.properties.resolution.enum
	metricResolutionValues = []string{"PER_JOB", "PERIOD"}
	// monetizationCategoryValues: components.schemas.JobTypeMonetization.properties.category.enum
	// Correction C2 — not named by D-27-11.
	monetizationCategoryValues = []string{"REVENUE", "COST_AVOIDED", "TIME_SAVED", "LEADING_VALUE"}
	// monetizationBasisValues: components.schemas.JobTypeMonetization.properties.basis.enum
	// Correction C2 — not named by D-27-11.
	monetizationBasisValues = []string{"REALIZED", "EXPECTED"}
)

// validateEnumField checks one closed-set field on container, reporting the
// JSON path and the full list of legal values on failure.
//
// An ABSENT or null field is not a violation: a closed set constrains the
// values a field may take, not whether the field appears. These schemas
// declare no `required` key at all, so required-ness goes to the API.
func validateEnumField(container map[string]interface{}, field, path string, legal []string) error {
	raw, present := container[field]
	if !present || raw == nil {
		return nil
	}

	value, isString := raw.(string)
	if !isString {
		return fmt.Errorf("%s must be a string (expected one of: %s)", path, strings.Join(legal, ", "))
	}

	for _, allowed := range legal {
		// Exact, case-sensitive byte equality — see the note on the enum
		// declarations above. No ToUpper, no EqualFold, no trimming.
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("%s is %q, which is not valid (expected one of: %s)", path, value, strings.Join(legal, ", "))
}

// validateEconomicsDocument checks the six closed enums and nothing else
// (D-27-11 plus correction C2), before any request is issued.
//
// This is the only client-side gate on six closed sets whose server-side
// rejection is a generic "Semantic validation failed" that need not name the
// legal values — and a typo inside a hand-edited JSON file is exactly what
// --help cannot catch.
//
// Every error names the JSON path INCLUDING its array index, because
// "invalid aggregation" without a position is nearly useless in a file
// carrying a dozen metric definitions.
//
// No other validation is applied. Ranges, required-ness and URL shape go to
// the API. Specifically: no 0-1 bound on any rate, and no USD rule on
// overheadCurrency or monetization.currency — the "Currently must be USD"
// text is a schema *description* on a property typed string rather than an
// enum, so enforcing it would invent a rule and could refuse input the API
// accepts. That is the same reasoning D-27-11 used to reject the rate bound.
func validateEconomicsDocument(doc map[string]interface{}) error {
	// A metrics value that is not an array, or an entry that is not an object,
	// is a shape error rather than an enum error — the API is the authority on
	// shape, so those travel on rather than being refused here.
	if metrics, ok := doc["metrics"].([]interface{}); ok {
		for i, rawEntry := range metrics {
			entry, ok := rawEntry.(map[string]interface{})
			if !ok {
				continue
			}

			path := fmt.Sprintf("metrics[%d]", i)
			checks := []struct {
				field string
				legal []string
			}{
				{"type", metricTypeValues},
				{"direction", metricDirectionValues},
				{"aggregation", metricAggregationValues},
				{"resolution", metricResolutionValues},
			}
			for _, check := range checks {
				if err := validateEnumField(entry, check.field, path+"."+check.field, check.legal); err != nil {
					return err
				}
			}
		}
	}

	// monetization is a oneOf object-or-null: an absent or null value fails
	// the type assertion and is correctly skipped.
	if monetization, ok := doc["monetization"].(map[string]interface{}); ok {
		if err := validateEnumField(monetization, "category", "monetization.category", monetizationCategoryValues); err != nil {
			return err
		}
		if err := validateEnumField(monetization, "basis", "monetization.basis", monetizationBasisValues); err != nil {
			return err
		}
	}

	return nil
}

// economicsDiff records what replacing the current contract with the next
// document would destroy.
//
// PUT replaces the whole document, and `set` must not pretend otherwise
// (D-27-04). This type is the input to the phase's principal safety control:
// a diff that under-reports loss makes the guard above it useless, so every
// class below is pinned by its own named test.
type economicsDiff struct {
	// LostMetricKeys and LostDimensionKeys are in current-document order, not
	// sorted, so the summary reads in the order the operator's own contract
	// declares them.
	LostMetricKeys    []string
	LostDimensionKeys []string
	// LostAllowedValues holds values dropped from dimensions that are
	// THEMSELVES retained, grouped by dimension key. A dropped dimension is
	// reported in LostDimensionKeys instead, not twice.
	LostAllowedValues map[string][]string
	// RedefinedMetricKeys names metrics that are RETAINED by key while a field
	// that defines what their number means is changed or dropped. In
	// current-document order.
	RedefinedMetricKeys []string
	// MonetizationRemoved is true when a non-null monetization object becomes
	// null or absent.
	MonetizationRemoved bool
	// RedefinedMonetizationFields names the fields of a RETAINED monetization
	// object that determine what the rule computes and are changed or dropped,
	// in schema order. Empty whenever MonetizationRemoved is true: a rule that
	// is gone is not a rule that changed.
	RedefinedMonetizationFields []string
	// UnitMetricKeyChanged is true when a non-empty anchor is re-pointed or
	// dropped. NewUnitMetricKey is empty in the dropped case.
	UnitMetricKeyChanged bool
	OldUnitMetricKey     string
	NewUnitMetricKey     string
	// UntrackableEntries counts, per collection, the entries of the CURRENT
	// contract that carry no string `key`. Such an entry cannot be matched
	// against the supplied document at all, so a replace over it cannot be
	// proven to lose nothing — and "cannot prove" is treated as loss here, the
	// same fail-closed reading the rest of this file adopts. Only the current
	// side is counted: an untrackable entry in the NEXT document puts nothing
	// declared at risk.
	UntrackableEntries map[string]int
}

// IsDestructive reports whether applying the next document would remove
// something the current contract declares.
func (d economicsDiff) IsDestructive() bool {
	return len(d.LostMetricKeys) > 0 ||
		len(d.LostDimensionKeys) > 0 ||
		len(d.LostAllowedValues) > 0 ||
		len(d.RedefinedMetricKeys) > 0 ||
		d.MonetizationRemoved ||
		len(d.RedefinedMonetizationFields) > 0 ||
		d.UnitMetricKeyChanged ||
		len(d.UntrackableEntries) > 0
}

// recordUntrackable notes that n entries of the current contract's named
// collection could not be matched against the supplied document. It is a no-op
// for n == 0 so the map stays nil — and therefore IsDestructive stays false —
// for the overwhelmingly common case of a well-formed contract.
func (d *economicsDiff) recordUntrackable(collection string, n int) {
	if n == 0 {
		return
	}
	if d.UntrackableEntries == nil {
		d.UntrackableEntries = map[string]int{}
	}
	d.UntrackableEntries[collection] = n
}

// Summary names each loss by key and by collection.
//
// This is the text an operator reads in a refusal, so "3 items would be lost"
// is not sufficient — the keys are the actionable content. Ordering is
// deterministic throughout: entries follow current-document order and the
// dimension keys of LostAllowedValues are sorted, because Go map iteration
// order is not stable and a refusal message that reshuffles between runs is
// hard to diff and hard to trust.
func (d economicsDiff) Summary() string {
	if !d.IsDestructive() {
		return "nothing declared in the current contract would be lost"
	}

	var parts []string
	if len(d.LostMetricKeys) > 0 {
		parts = append(parts, fmt.Sprintf("metrics that would be lost: %s", quotedList(d.LostMetricKeys)))
	}
	if len(d.RedefinedMetricKeys) > 0 {
		parts = append(parts, fmt.Sprintf("metrics that would be redefined: %s", quotedList(d.RedefinedMetricKeys)))
	}
	if len(d.LostDimensionKeys) > 0 {
		parts = append(parts, fmt.Sprintf("dimensions that would be lost: %s", quotedList(d.LostDimensionKeys)))
	}
	if len(d.LostAllowedValues) > 0 {
		dimensionKeys := make([]string, 0, len(d.LostAllowedValues))
		for key := range d.LostAllowedValues {
			dimensionKeys = append(dimensionKeys, key)
		}
		sort.Strings(dimensionKeys)
		for _, key := range dimensionKeys {
			parts = append(parts, fmt.Sprintf("allowed values that would be lost from dimension %q: %s",
				key, quotedList(d.LostAllowedValues[key])))
		}
	}
	if d.MonetizationRemoved {
		parts = append(parts, "the monetization rule would be removed")
	}
	if len(d.RedefinedMonetizationFields) > 0 {
		parts = append(parts, fmt.Sprintf("the monetization rule would be redefined (%s)",
			strings.Join(d.RedefinedMonetizationFields, ", ")))
	}
	// Untrackable entries are reported LAST and in sorted collection order, so
	// the concrete losses an operator can act on come first and the ordering is
	// stable between runs.
	if len(d.UntrackableEntries) > 0 {
		collections := make([]string, 0, len(d.UntrackableEntries))
		for collection := range d.UntrackableEntries {
			collections = append(collections, collection)
		}
		sort.Strings(collections)
		for _, collection := range collections {
			n := d.UntrackableEntries[collection]
			verb := "declare"
			if n == 1 {
				verb = "declares"
			}
			parts = append(parts, fmt.Sprintf(
				"%s in the current contract's %q %s no key and cannot be matched against the "+
					"supplied document, so this replace cannot be proven safe",
				pluralEntries(n), collection, verb))
		}
	}
	if d.UnitMetricKeyChanged {
		if d.NewUnitMetricKey == "" {
			parts = append(parts, fmt.Sprintf("the unit metric key %q would be removed", d.OldUnitMetricKey))
		} else {
			parts = append(parts, fmt.Sprintf("the unit metric key would change from %q to %q",
				d.OldUnitMetricKey, d.NewUnitMetricKey))
		}
	}

	return strings.Join(parts, "; ")
}

// pluralEntries renders an entry count with the matching noun. "1 entries" in
// a refusal message reads as a bug in the tool and undermines the sentence it
// appears in.
func pluralEntries(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

// quotedList renders keys as a quoted, comma-separated list so a key
// containing a space or an empty key is still legible in a refusal message.
func quotedList(keys []string) string {
	quoted := make([]string, 0, len(keys))
	for _, key := range keys {
		quoted = append(quoted, fmt.Sprintf("%q", key))
	}
	return strings.Join(quoted, ", ")
}

// indexEconomicsCollection returns the `key` strings of doc[collection] in
// document order, the entries indexed by that key, and the number of entries it
// could not index at all.
//
// A collection that is absent, null or not an array yields nothing — shape is
// the API's authority, not ours.
//
// An entry that is not an object, or that carries no string `key`, cannot be
// tracked across documents and is therefore counted rather than silently
// dropped. Counting it is the whole point: a skipped entry is invisible to the
// diff in BOTH documents, so without the count, deleting one registers as no
// loss and the caller states — falsely — that nothing would be lost. The
// caller decides what to do with the count; on the CURRENT side it means the
// replace cannot be proven safe.
//
// EVERY entry under a key is kept, in document order, rather than the last one
// winning. Nothing rejects a duplicate key — the schemas declare no uniqueness
// constraint and validateEconomicsDocument does not look at keys at all — so a
// hand-edited file that duplicates a dimension (a plausible copy-paste result)
// round-trips into the server. Keeping only the last entry made everything the
// earlier duplicate declared invisible, and its loss was then reported as no
// loss. The caller resolves the duplicates in whichever direction is
// fail-closed for the side it is reading.
func indexEconomicsCollection(doc map[string]interface{}, collection string) ([]string, map[string][]map[string]interface{}, int) {
	order := []string{}
	byKey := map[string][]map[string]interface{}{}
	untrackable := 0

	entries, ok := doc[collection].([]interface{})
	if !ok {
		return order, byKey, untrackable
	}

	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			untrackable++
			continue
		}
		key, ok := entry["key"].(string)
		if !ok {
			untrackable++
			continue
		}
		if _, seen := byKey[key]; !seen {
			order = append(order, key)
		}
		byKey[key] = append(byKey[key], entry)
	}

	return order, byKey, untrackable
}

// allowedValuesOf returns a dimension's allowedValues in document order, plus
// a membership set.
func allowedValuesOf(dimension map[string]interface{}) ([]string, map[string]bool) {
	order := []string{}
	set := map[string]bool{}

	values, ok := dimension["allowedValues"].([]interface{})
	if !ok {
		return order, set
	}

	for _, rawValue := range values {
		value, ok := rawValue.(string)
		if !ok {
			continue
		}
		if !set[value] {
			order = append(order, value)
		}
		set[value] = true
	}

	return order, set
}

// allowedValuesUnion returns, in document order, every value declared by ANY of
// entries. It is how the CURRENT side resolves duplicate keys: a value the
// operator declared anywhere under that key is a value they declared, and it
// must survive the replace or be reported as lost.
func allowedValuesUnion(entries []map[string]interface{}) []string {
	order := []string{}
	seen := map[string]bool{}
	for _, entry := range entries {
		values, _ := allowedValuesOf(entry)
		for _, value := range values {
			if seen[value] {
				continue
			}
			seen[value] = true
			order = append(order, value)
		}
	}
	return order
}

// allowedValuesIntersection returns the values declared by EVERY one of entries.
// It is how the NEXT side resolves duplicate keys: we cannot know which
// duplicate the server will honour, so a value is credited as surviving only
// when it survives whichever one wins. With the single entry of a well-formed
// document this is exactly that entry's values, so the ordinary case is
// unaffected.
func allowedValuesIntersection(entries []map[string]interface{}) map[string]bool {
	if len(entries) == 0 {
		return map[string]bool{}
	}
	_, common := allowedValuesOf(entries[0])
	for _, entry := range entries[1:] {
		_, set := allowedValuesOf(entry)
		for value := range common {
			if !set[value] {
				delete(common, value)
			}
		}
	}
	return common
}

// metricMeaningFields are the fields of a JobTypeMetricDefinition that define
// what the metric's number MEANS rather than how it is presented: a metric that
// keeps its key while changing them is a different measurement wearing the same
// name. All four are closed enums (see the enum block above).
//
// `key` is absent because it is the identity the diff matches on — a changed
// key is a lost metric plus an added one, which the collection diff already
// reports.
var metricMeaningFields = []string{"type", "direction", "aggregation", "resolution"}

// monetizationMeaningFields are the fields of a JobTypeMonetization that
// determine what the value rule computes and how the result is booked.
//
// valuePerUnit and currency are deliberately EXCLUDED. They are scalar edits
// whose replaced value is visible in the round-trip, which is the same reason
// unitLabel, overheadPerUnit and overheadCurrency are not destructive. This
// boundary is recorded here rather than left implicit, because the asymmetry
// it replaces — metricKey re-pointed silently while unitMetricKey re-pointed
// requires --yes — was a classification gap, not a considered decision.
var monetizationMeaningFields = []string{"metricKey", "category", "basis"}

// redefinedFields returns those of fields that the current entries declare and
// the next entries do not preserve, in the order given.
//
// The comparison rule is the SAME one diffEconomics applies to unitMetricKey,
// and deliberately so: a value the current contract does not declare is
// empty-to-anything and loses nothing (the create case), while a declared value
// that is changed OR omitted is a loss. A non-string value reads as empty for
// the same reason it does there — shape is the API's authority.
//
// Duplicates on either side resolve toward reporting loss: every current entry
// that declares the field must be matched by EVERY next entry, since we cannot
// know which duplicate the server honours.
func redefinedFields(currentEntries, nextEntries []map[string]interface{}, fields []string) []string {
	var redefined []string
	for _, field := range fields {
		if fieldRedefined(currentEntries, nextEntries, field) {
			redefined = append(redefined, field)
		}
	}
	return redefined
}

func fieldRedefined(currentEntries, nextEntries []map[string]interface{}, field string) bool {
	for _, currentEntry := range currentEntries {
		declared, _ := currentEntry[field].(string)
		if declared == "" {
			continue
		}
		for _, nextEntry := range nextEntries {
			if supplied, _ := nextEntry[field].(string); supplied != declared {
				return true
			}
		}
	}
	return false
}

// diffEconomics computes what replacing current with next would destroy.
//
// Entry identity is EXACT byte equality of the UTF-8 `key` string — no case
// folding, no trimming, no Unicode normalization. `Region` and `region` are
// two different dimensions, and an NFD-decomposed key is not the same key as
// its NFC form. This is the same equality rule validateEconomicsDocument uses
// for enum membership, and it is stated here so a later "helpful"
// normalization cannot silently merge two declared dimensions into one.
//
// Classification follows D-27-04 and RESEARCH Pattern 4:
//
//   - a dropped metrics entry, a dropped dimensions entry: destructive,
//     explicit in D-27-04;
//   - a dropped allowedValues entry inside a RETAINED dimension: destructive —
//     the same class of loss one level down, since a declared value silently
//     stops being accepted;
//   - monetization going from an object to null or absent: destructive — it
//     deletes the entire value-derivation rule and nothing else records it;
//   - a changed or dropped non-empty unitMetricKey: destructive. This is the
//     discretionary call CONTEXT left open. It is the anchor every baseline's
//     per-unit figure is denominated in, so re-pointing it silently makes
//     historical baselines mean something different. Erring toward
//     confirmation costs one extra --yes; erring the other way is expensive to
//     discover later. D-27-04 is marked reversible, so this is trivially
//     removable if operators find it noisy;
//   - a metrics entry that is RETAINED by key while its type, direction,
//     aggregation or resolution is changed or dropped: destructive. This is the
//     unitMetricKey argument applied one level down — aggregation and
//     resolution define what the number means, so a metric that keeps its name
//     while changing them makes every historical reading of that name mean
//     something else. Classifying a re-pointed unitMetricKey as destructive and
//     a re-typed metric as free was a gap, not a boundary;
//   - a monetization object that is RETAINED while its metricKey, category or
//     basis is changed or dropped: destructive, for the same reason —
//     metricKey selects which metric the value rule reads. valuePerUnit and
//     currency are excluded as scalar edits (see monetizationMeaningFields);
//   - duplicate entry keys are resolved toward loss, never away from it: the
//     current side unions what the duplicates declare and the next side keeps
//     only what all of them agree on. Nothing rejects a duplicate key, so this
//     is a real input, and letting the last entry win made the earlier one's
//     declarations vanish unreported;
//   - an entry of the CURRENT contract that carries no string `key`, or that is
//     not an object at all: destructive. It cannot be matched against the
//     supplied document in either direction, so the tool cannot prove nothing
//     is lost — and "cannot prove" is treated as loss, which is the same
//     fail-closed posture the rest of this file adopts. The same entry in the
//     NEXT document is ignored: nothing declared is at risk;
//   - additions are never destructive, and unitLabel, overheadPerUnit and
//     overheadCurrency edits are not: they are scalar changes whose replaced
//     value is visible in the round-trip;
//   - an empty or absent current contract yields a non-destructive diff —
//     nothing exists to lose, which is the create case `set` must support.
func diffEconomics(current, next map[string]interface{}) economicsDiff {
	var diff economicsDiff

	currentMetricOrder, currentMetrics, untrackableMetrics := indexEconomicsCollection(current, "metrics")
	_, nextMetrics, _ := indexEconomicsCollection(next, "metrics")
	diff.recordUntrackable("metrics", untrackableMetrics)
	for _, key := range currentMetricOrder {
		nextEntries := nextMetrics[key]
		if len(nextEntries) == 0 {
			diff.LostMetricKeys = append(diff.LostMetricKeys, key)
			continue
		}
		// The key survives, but the measurement behind it may not.
		if len(redefinedFields(currentMetrics[key], nextEntries, metricMeaningFields)) > 0 {
			diff.RedefinedMetricKeys = append(diff.RedefinedMetricKeys, key)
		}
	}

	currentDimensionOrder, currentDimensions, untrackableDimensions := indexEconomicsCollection(current, "dimensions")
	_, nextDimensions, _ := indexEconomicsCollection(next, "dimensions")
	diff.recordUntrackable("dimensions", untrackableDimensions)
	for _, key := range currentDimensionOrder {
		nextEntries := nextDimensions[key]
		if len(nextEntries) == 0 {
			// The whole dimension is gone; its allowed values are reported
			// with it rather than listed a second time.
			diff.LostDimensionKeys = append(diff.LostDimensionKeys, key)
			continue
		}

		// Union on the current side, intersection on the next: both directions
		// resolve a duplicate key toward reporting loss rather than hiding it.
		currentValues := allowedValuesUnion(currentDimensions[key])
		nextValues := allowedValuesIntersection(nextEntries)
		var lost []string
		for _, value := range currentValues {
			if !nextValues[value] {
				lost = append(lost, value)
			}
		}
		if len(lost) > 0 {
			if diff.LostAllowedValues == nil {
				diff.LostAllowedValues = map[string][]string{}
			}
			diff.LostAllowedValues[key] = lost
		}
	}

	// monetization is a oneOf object-or-null. The assertion fails for both
	// null and absent, which are the same loss from the operator's side.
	if currentMonetization, declared := current["monetization"].(map[string]interface{}); declared {
		nextMonetization, retained := next["monetization"].(map[string]interface{})
		if !retained {
			diff.MonetizationRemoved = true
		} else {
			// Retained as an object, but possibly pointed at a different
			// metric or booked a different way. Reported instead of
			// MonetizationRemoved, never alongside it: a rule that is gone is
			// not a rule that changed.
			diff.RedefinedMonetizationFields = redefinedFields(
				[]map[string]interface{}{currentMonetization},
				[]map[string]interface{}{nextMonetization},
				monetizationMeaningFields)
		}
	}

	// A missing or non-string unitMetricKey reads as empty. Empty-to-anything
	// is the create case and loses nothing; anything-to-different is a
	// re-anchoring, and anything-to-empty is a strictly larger loss than that,
	// so both are flagged.
	currentUnitMetricKey, _ := current["unitMetricKey"].(string)
	nextUnitMetricKey, _ := next["unitMetricKey"].(string)
	if currentUnitMetricKey != "" && currentUnitMetricKey != nextUnitMetricKey {
		diff.UnitMetricKeyChanged = true
		diff.OldUnitMetricKey = currentUnitMetricKey
		diff.NewUnitMetricKey = nextUnitMetricKey
	}

	return diff
}
