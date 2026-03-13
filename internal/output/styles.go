package output

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// borderStyle uses a subtle gray foreground for table borders.
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// headerStyle renders table headers in bold with an accent color.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(0, 1)

	// cellStyle renders plain data cells with padding.
	cellStyle = lipgloss.NewStyle().Padding(0, 1)

	// Semantic cell styles keyed by column role.
	idStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("73")).Padding(0, 1)
	nameStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Padding(0, 1)
	categoryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Padding(0, 1)
	quantityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
	moneyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Padding(0, 1)
	dateStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Padding(0, 1)
	boolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Padding(0, 1)
)

// columnRole maps header names to semantic styles.
// This ensures consistent coloring of the same data type across all commands.
var columnRole = map[string]lipgloss.Style{
	// Identity
	"ID":       idStyle,
	"Alert ID": idStyle,
	"Trace ID": idStyle,

	// Names / labels
	"Name":    nameStyle,
	"Label":   nameStyle,
	"Setting": nameStyle,

	// Categories / classifications
	"Provider": categoryStyle,
	"Type":     categoryStyle,
	"Mode":     categoryStyle,
	"Source":   categoryStyle,
	"Tool":     categoryStyle,
	"Model":    categoryStyle,
	"Roles":    categoryStyle,

	// Quantities
	"Tokens":       quantityStyle,
	"Total Tokens": quantityStyle,
	"Count":        quantityStyle,
	"Duration":     quantityStyle,
	"Invocations":  quantityStyle,
	"Requests":     quantityStyle,
	"Executions":   quantityStyle,
	"Entries":      quantityStyle,

	// Money
	"Cost":       moneyStyle,
	"Total Cost": moneyStyle,
	"Price":      moneyStyle,
	"Budget":     moneyStyle,
	"Current":    moneyStyle,
	"Remaining":  moneyStyle,

	// Percentages / metrics
	"% Used":  quantityStyle,
	"Latency": quantityStyle,

	// Dates
	"Created": dateStyle,

	// Secondary / dim
	"Description": dimStyle,
	"Email":       dimStyle,
	"Secret":      dimStyle,
	"Value":       dimStyle,

	// Errors
	"Errors": errorStyle,

	// Boolean
	"Enabled": boolStyle,
}

// cellStyleForHeader returns the appropriate style for a cell based on its
// column header name. Falls back to the default cellStyle for unknown headers.
func cellStyleForHeader(header string) lipgloss.Style {
	if s, ok := columnRole[header]; ok {
		return s
	}
	return cellStyle
}

// statusStyle returns a style colored by status value.
// "active"/"enabled" -> green, "inactive"/"disabled"/"deleted" -> red,
// "pending"/"draft" -> yellow, default -> plain.
func statusStyle(status string) lipgloss.Style {
	base := lipgloss.NewStyle().Padding(0, 1)
	switch strings.ToLower(status) {
	case "active", "enabled", "true":
		return base.Foreground(lipgloss.Color("42")) // green
	case "inactive", "disabled", "deleted", "false":
		return base.Foreground(lipgloss.Color("196")) // red
	case "pending", "draft":
		return base.Foreground(lipgloss.Color("214")) // yellow
	default:
		return base
	}
}
