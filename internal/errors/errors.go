// Package errors provides styled error rendering and API error types.
package errors

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// APIError represents an error response from the Revenium API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

// Error implements the error interface, returning the error message.
func (e *APIError) Error() string {
	return e.Message
}

// VerboseError returns a detailed error string including status code and response body.
func (e *APIError) VerboseError() string {
	return fmt.Sprintf("%s (status: %d, body: %s)", e.Message, e.StatusCode, e.Body)
}

// RenderError returns the message wrapped in a Lip Gloss styled error box
// with a red rounded border.
func RenderError(msg string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Foreground(lipgloss.Color("196")).
		Padding(0, 1)

	return style.Render("Error: " + msg)
}
