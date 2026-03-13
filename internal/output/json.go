package output

import (
	"encoding/json"
)

// RenderJSON writes data as pretty-printed JSON (2-space indent) to the
// Formatter's writer. It handles both arrays (slices) and single objects
// (structs/maps) since json.Encoder handles both.
func (f *Formatter) RenderJSON(data interface{}) error {
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderJSONError writes a JSON error object to the Formatter's error writer.
// The output shape is {"error": "msg", "exit_code": N, "status": N} with 2-space indent.
// Always writes to errWriter (even in quiet mode -- errors always go to stderr).
func (f *Formatter) RenderJSONError(msg string, statusCode int, exitCode int) error {
	errObj := map[string]interface{}{
		"error":     msg,
		"status":    statusCode,
		"exit_code": exitCode,
	}
	enc := json.NewEncoder(f.errWriter)
	enc.SetIndent("", "  ")
	return enc.Encode(errObj)
}

// Render is a convenience method that dispatches to RenderJSON or RenderTable
// based on the Formatter's mode. Resource commands should call this method.
// If jsonMode is active, data is rendered as JSON; otherwise, the table
// definition and rows are rendered as a styled table.
// When fields are set via SetFields, both JSON and table output are filtered
// to include only the specified fields.
func (f *Formatter) Render(def TableDef, rows [][]string, data interface{}) error {
	if f.jsonMode {
		return f.RenderJSON(FilterFields(data, f.fields))
	}
	filteredDef, filteredRows := FilterTableDef(def, rows, f.fields)
	return f.RenderTable(filteredDef, filteredRows)
}
