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
)

// statusStyle returns a style colored by status value.
// "active"/"enabled" -> green, "inactive"/"disabled"/"deleted" -> red,
// "pending"/"draft" -> yellow, default -> plain.
func statusStyle(status string) lipgloss.Style {
	base := lipgloss.NewStyle().Padding(0, 1)
	switch strings.ToLower(status) {
	case "active", "enabled":
		return base.Foreground(lipgloss.Color("42")) // green
	case "inactive", "disabled", "deleted":
		return base.Foreground(lipgloss.Color("196")) // red
	case "pending", "draft":
		return base.Foreground(lipgloss.Color("214")) // yellow
	default:
		return base
	}
}
