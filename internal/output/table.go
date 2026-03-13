package output

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// TableDef defines the structure for a table to be rendered.
type TableDef struct {
	// Headers are the column header strings.
	Headers []string
	// StatusColumn is the index of the status column for color styling.
	// Set to -1 if no status column exists.
	StatusColumn int
}

// RenderTable renders rows as a styled table with rounded borders.
// In quiet mode (and not JSON mode), it returns nil without producing output.
// The writer handles ANSI stripping for non-TTY and NO_COLOR environments.
func (f *Formatter) RenderTable(def TableDef, rows [][]string) error {
	if f.quiet && !f.jsonMode {
		return nil
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers(def.Headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			// Status columns use value-based dynamic coloring
			if col == def.StatusColumn && row >= 0 && row < len(rows) {
				return statusStyle(rows[row][col])
			}
			// All other columns use header-name-based semantic coloring
			if col >= 0 && col < len(def.Headers) {
				header := def.Headers[col]
				// "Enabled" and "Risk" columns also get value-based status coloring
				if header == "Enabled" || header == "Risk" {
					if row >= 0 && row < len(rows) {
						return statusStyle(rows[row][col])
					}
				}
				return cellStyleForHeader(header)
			}
			return cellStyle
		})

	_, err := fmt.Fprintln(f.writer, t.String())
	return err
}

// Truncate truncates s to maxLen runes, appending a Unicode ellipsis (U+2026)
// if truncation occurs. Uses rune-based truncation for correct Unicode handling.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "\u2026"
}
