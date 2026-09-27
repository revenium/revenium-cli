package jobs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

// baselinesCmd is the parent `baselines` sub-resource under `jobs types`
// (D-27-07).
//
// This file declares NO package initializer. Go runs package initializers in
// lexical filename order, and baselines.go sorts before types.go, so an
// initializer here would run first and see a nil typesCmd — registering
// nothing, or panicking on every `revenium` invocation including --help.
// Registration therefore lives in types.go and calls initBaselines() below,
// mirroring economics.go and cmd/teams/prompt_capture.go.
//
// The help text below deliberately never says a baseline can be changed. The
// API models baselines as append-only, and CONTEXT asks that the append-only
// nature be visible in the interface rather than merely true of the API — an
// operator who reads "edit" here would form a belief the platform will not
// honour.
var baselinesCmd = &cobra.Command{
	Use:   "baselines",
	Short: "Read and append a job type's immutable pre-AI baseline versions",
	Long: "Read and append a job type's immutable pre-AI baseline versions.\n\n" +
		"Each baseline version is permanent. Appending a new version leaves every\n" +
		"earlier version readable and unchanged; the newest version is the one in\n" +
		"force, and `list` shows it first.",
	Example: `  # List a job type's baseline versions, newest first
  revenium jobs types baselines list acme-review

  # Append a new baseline version
  revenium jobs types baselines append acme-review --cost-per-unit 4.25 --minutes-per-unit 18 --effective-from 2026-01-01T00:00:00Z

  # As JSON for scripting
  revenium jobs types baselines list acme-review --json`,
}

// initBaselines registers the baselines subcommands. Called from types.go's
// package initializer after typesCmd is assigned — see the comment on
// baselinesCmd for why it is not called from this file.
func initBaselines() {
	baselinesCmd.AddCommand(newBaselinesListCmd())
	baselinesCmd.AddCommand(newBaselinesAppendCmd())
}

// baselinesTableDef is the row-per-version layout shared by `baselines list`
// and the single-row render `baselines append` emits for the version it just
// appended, so the two surfaces cannot drift into different column sets.
//
// StatusColumn: -1 disables status colorization — no cell here carries status
// semantics. The table surfaces eight of BaselineResource's eleven properties;
// all eleven remain reachable under --json, which receives the untouched
// server payload.
var baselinesTableDef = output.TableDef{
	Headers: []string{
		"Version", "Cost/Unit", "Minutes/Unit", "Quality Rate",
		"Hourly Rate", "Provenance", "Declared By", "Effective From",
	},
	StatusColumn: -1,
}

// toBaselineRows builds the table rows for a decoded slice of
// BaselineResource objects.
//
// Row order is the caller's: this helper preserves the slice as given and
// performs no sorting of its own, so `baselines list` can hand the same sorted
// slice to both this function and Render's third argument.
//
// Cell order is INTENTIONALLY an explicit literal rather than a range over the
// response map, which would produce non-deterministic Go map iteration order.
//
// Every numeric cell goes through money() or num(), never str(): str() uses
// the unformatted fmt print family, which emits 1.234567e+06 for any
// JSON-decoded number at or above 1e6. Each row's currency is read from that
// row's own `currency` key — baselines are versioned independently, so a
// per-slice currency would misattribute an earlier version's denomination.
func toBaselineRows(baselines []map[string]interface{}) [][]string {
	rows := make([][]string, len(baselines))
	for i, b := range baselines {
		currency := str(b, "currency")
		rows[i] = []string{
			num(b, "version", "%.0f"),
			money(b, "costPerUnit", currency),
			num(b, "minutesPerUnit", "%.2f"),
			num(b, "qualityRate", "%.4f"),
			money(b, "hourlyRate", currency),
			str(b, "provenance"),
			str(b, "declaredBy"),
			str(b, "effectiveFrom"),
		}
	}
	return rows
}

// requireEffectiveFrom refuses, before any HTTP request is issued, when
// --effective-from was not supplied to `baselines append` (G-1, D11).
//
// Why a client-side guard rather than letting the server answer: the dev host
// rejects the omission with HTTP 400 carrying the bare body `effectiveFrom`,
// which tells the operator nothing about which flag to add. Every other guard
// in this phase refuses locally and names the flag; requireTeam() sets that
// standard for --team-id under SC5.
//
// Why this is not inferable from the schema: BaselineRequest declares no
// `required` key at all, and effectiveFrom carries no documented default, so
// the field's required-ness was undecidable until a live call settled it
// (recorded as D11 from plan 27-03, answered 2026-09-05).
func requireEffectiveFrom(c *cobra.Command) error {
	if !c.Flags().Changed("effective-from") {
		return fmt.Errorf(
			"no baseline effective date: this command requires --effective-from.\n" +
				"Supply an RFC 3339 timestamp, for example: --effective-from 2026-01-07T00:00:00Z")
	}
	return nil
}
