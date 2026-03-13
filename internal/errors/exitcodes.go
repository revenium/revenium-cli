package errors

import (
	"errors"
	"net"
)

// Semantic exit codes for AI agent and script consumption.
const (
	ExitOK         = 0 // success
	ExitGeneral    = 1 // general/unknown error
	ExitAuth       = 2 // 401/403
	ExitNotFound   = 3 // 404
	ExitValidation = 4 // 400/422
	ExitNetwork    = 5 // connection failures
)

// ExitCodeFor maps an error to a semantic exit code.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			return ExitAuth
		case apiErr.StatusCode == 404:
			return ExitNotFound
		case apiErr.StatusCode == 400 || apiErr.StatusCode == 422:
			return ExitValidation
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ExitNetwork
	}

	return ExitGeneral
}
