package jobs

import (
	"fmt"
	"io"
	"strings"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// economicsScalarsTableDef is the 2-column key-value layout for the contract
// scalars — the cmd/teams/prompt_capture.go renderPromptSettings shape D-27-09
// names. A status column index of -1 disables status colorization; none of
// these Value cells carry status semantics, and neither do the other three
// sections below.
var economicsScalarsTableDef = output.TableDef{
	Headers:      []string{"Setting", "Value"},
	StatusColumn: -1,
}

// metricsTableDef is the JobTypeMetricDefinition section: one row per declared
// metric, one column per schema property.
var metricsTableDef = output.TableDef{
	Headers:      []string{"Key", "Type", "Direction", "Aggregation", "Resolution"},
	StatusColumn: -1,
}

// dimensionsTableDef is the JobTypeDimensionDefinition section. allowedValues
// is a bare string array, joined into one cell rather than exploded into rows.
var dimensionsTableDef = output.TableDef{
	Headers:      []string{"Key", "Allowed Values"},
	StatusColumn: -1,
}

// currentBaselineTableDef is the BaselineResource section, rendered key-value
// because a single baseline in eleven columns is unreadable in a terminal.
var currentBaselineTableDef = output.TableDef{
	Headers:      []string{"Field", "Value"},
	StatusColumn: -1,
}

// renderEconomics renders a JobTypeEconomicsResource as D-27-09's four labelled
// sections — Contract, Metrics, Dimensions, Current Baseline, in that fixed
// order — or, in JSON mode, as the server's resource untouched.
//
// WHY THIS FILE BRANCHES ON OUTPUT MODE WHEN cmd/jobs/roi.go SAYS NOT TO.
// roi.go's rule (a render helper must not branch on JSON mode) is correct for a
// single-table render: Formatter.Render dispatches table-vs-JSON internally, so
// a second branch there would be redundant and would drift. A FOUR-section
// render cannot use that dispatch, because calling Render once per section
// emits four separate JSON documents in JSON mode — and passing a nil payload
// on three of them emits three JSON nulls. So this function opens with exactly
// ONE branch: JSON mode hands the whole untouched resource to Render once and
// returns. Routing it through Render rather than RenderJSON keeps --fields
// filtering working exactly as it does everywhere else in the repo.
//
// Everything below that branch is the table path by construction. No section,
// and no empty-state line, may consult output mode a second time — a second
// branch is how the JSON path and the table path drift apart.
//
// Pitfall 10: --json is verbatim, but `--json --fields x,y` is filtered by
// Formatter.Render (internal/output/json.go:36-42). So the D-27-03 round-trip
// holds for `get --json` and NOT for `get --json --fields`. That is correct,
// existing, repo-wide behaviour and needs no fix — it is recorded here so it is
// documented rather than later discovered as a bug.
//
// The resource is never mutated before being handed to Render, and it is
// decoded as a map rather than a typed struct, so a server field this code does
// not know about still round-trips through `get --json > f.json`.
func renderEconomics(resource map[string]interface{}) error {
	if cmd.Output.IsJSON() {
		return cmd.Output.Render(economicsScalarsTableDef, nil, resource)
	}

	// Sections render through RenderTable directly, which already returns
	// without output in quiet mode (internal/output/table.go). Headings go to
	// the Formatter's writer, which is io.Discard in quiet mode.
	w := cmd.Output.Writer()

	fmt.Fprintln(w, "Contract")
	if err := cmd.Output.RenderTable(economicsScalarsTableDef, economicsScalarRows(resource)); err != nil {
		return err
	}

	fmt.Fprintln(w, "Metrics")
	if err := renderMetricsSection(w, resource); err != nil {
		return err
	}

	fmt.Fprintln(w, "Dimensions")
	if err := renderDimensionsSection(w, resource); err != nil {
		return err
	}

	fmt.Fprintln(w, "Current Baseline")
	return renderCurrentBaselineSection(w, resource)
}

// economicsScalarRows builds the Contract section.
//
// Row order is INTENTIONALLY an explicit ordered slice literal. A for/range
// over the response map would produce non-deterministic Go map iteration order
// — the pitfall cmd/jobs/roi.go:66-70 already records for a prior phase.
//
// Correction C1: overheadCurrency is a real seventh JobTypeEconomicsRequest
// property that neither the ROADMAP's JOBS-12 wording nor D-27-01's field list
// mentions. It both drives the Overhead Per Unit currency prefix and renders as
// its own row, so it is visible rather than silently dropped.
//
// The nested JobTypeMonetization object is flattened into this section rather
// than given a section of its own: it is at most one object, and a fifth
// heading over five rows reads as noise.
func economicsScalarRows(resource map[string]interface{}) [][]string {
	currency := str(resource, "overheadCurrency")
	rows := [][]string{
		{"Job Type", str(resource, "jobType")},
		{"Unit Metric Key", str(resource, "unitMetricKey")},
		{"Unit Label", str(resource, "unitLabel")},
		{"Overhead Per Unit", money(resource, "overheadPerUnit", currency)},
		{"Overhead Currency", currency},
	}

	monetization, ok := objectAt(resource, "monetization")
	if !ok {
		// monetization is oneOf[JobTypeMonetization, null]. Absent or null is
		// one fact about the contract, not five blank rows.
		return append(rows, []string{"Monetization", "not declared"})
	}

	// valuePerUnit is denominated in the monetization object's OWN currency,
	// which is not required to equal overheadCurrency.
	monetizationCurrency := str(monetization, "currency")
	return append(rows,
		[]string{"Monetization Metric Key", str(monetization, "metricKey")},
		[]string{"Monetization Value Per Unit", money(monetization, "valuePerUnit", monetizationCurrency)},
		[]string{"Monetization Currency", monetizationCurrency},
		[]string{"Monetization Category", str(monetization, "category")},
		[]string{"Monetization Basis", str(monetization, "basis")},
	)
}

// renderMetricsSection renders one row per JobTypeMetricDefinition. All five
// properties are strings, so str() is correct here — there is no numeric cell
// in this section.
//
// The empty state is deliberate, not blank (D-27-09): a header-only table and
// an absent section are indistinguishable to an operator, and only one of them
// is a fact about the contract. This branch sits below renderEconomics's single
// top-of-function dispatch and therefore runs on the table path by
// construction — it must not consult output mode again. The JSON-mode arm has
// already returned the complete resource by the time it is reachable, so an
// empty section changes the rendered table and never the emitted document.
func renderMetricsSection(w io.Writer, resource map[string]interface{}) error {
	metrics := objectsAt(resource, "metrics")
	if len(metrics) == 0 {
		_, err := fmt.Fprintln(w, "No metrics are declared for this job type.")
		return err
	}
	rows := make([][]string, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, []string{
			str(metric, "key"),
			str(metric, "type"),
			str(metric, "direction"),
			str(metric, "aggregation"),
			str(metric, "resolution"),
		})
	}
	return cmd.Output.RenderTable(metricsTableDef, rows)
}

// renderDimensionsSection renders one row per JobTypeDimensionDefinition, with
// allowedValues joined into a single cell. Its empty state carries the same
// reasoning as renderMetricsSection's.
func renderDimensionsSection(w io.Writer, resource map[string]interface{}) error {
	dimensions := objectsAt(resource, "dimensions")
	if len(dimensions) == 0 {
		_, err := fmt.Fprintln(w, "No dimensions are declared for this job type.")
		return err
	}
	rows := make([][]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		rows = append(rows, []string{
			str(dimension, "key"),
			strings.Join(stringsAt(dimension, "allowedValues"), ", "),
		})
	}
	return cmd.Output.RenderTable(dimensionsTableDef, rows)
}

// renderCurrentBaselineSection renders the eleven BaselineResource properties
// as an explicit ordered key-value list.
//
// Every numeric cell routes through money() or num(); this file declares no
// numeric formatter of its own. str() must never reach a numeric field — the
// unformatted no-verb print variant it uses emits scientific notation at or
// above 1e6, and output.FloatVal alone cannot tell an undeclared value from a
// declared zero.
//
// costPerUnit and hourlyRate are denominated in the baseline's OWN currency,
// not the contract's overheadCurrency: a baseline is an immutable version and
// carries the currency it was declared in.
// A contract with no baseline yet is the most consequential of the three empty
// states: every downstream saving is computed against a baseline, so "none
// declared" must read as a fact and name the command that fixes it.
func renderCurrentBaselineSection(w io.Writer, resource map[string]interface{}) error {
	baseline, ok := objectAt(resource, "currentBaseline")
	if !ok {
		_, err := fmt.Fprintln(w,
			"No baseline version has been declared yet. "+
				"Declare one with `jobs types baselines append`.")
		return err
	}
	currency := str(baseline, "currency")
	rows := [][]string{
		{"Version", num(baseline, "version", "%.0f")},
		{"Cost Per Unit", money(baseline, "costPerUnit", currency)},
		{"Minutes Per Unit", num(baseline, "minutesPerUnit", "%.2f")},
		{"Quality Rate", num(baseline, "qualityRate", "%.4f")},
		{"Hourly Rate", money(baseline, "hourlyRate", currency)},
		{"Currency", currency},
		{"Provenance", str(baseline, "provenance")},
		{"Declared By", str(baseline, "declaredBy")},
		{"Evidence URL", str(baseline, "evidenceUrl")},
		{"Effective From", str(baseline, "effectiveFrom")},
		{"Created", str(baseline, "created")},
	}
	return cmd.Output.RenderTable(currentBaselineTableDef, rows)
}

// objectAt returns the nested object at key. The second result is false when
// the key is absent, JSON null, or not an object — the three shapes a
// oneOf[Schema, null] property can arrive in.
func objectAt(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	nested, ok := m[key].(map[string]interface{})
	return nested, ok
}

// objectsAt returns the array of objects at key, or nil when the key is
// absent, JSON null, or an empty array.
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

// stringsAt returns the array of strings at key, skipping any non-string entry
// rather than rendering Go's default formatting for it.
func stringsAt(m map[string]interface{}, key string) []string {
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		if value, ok := entry.(string); ok {
			values = append(values, value)
		}
	}
	return values
}
