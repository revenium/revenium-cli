// Package validate provides input validation for CLI arguments.
package validate

import (
	"fmt"
	"strings"
)

// ResourceID validates a resource ID argument, rejecting values that contain
// control characters, embedded query parameters, path traversal sequences,
// or pre-encoded sequences. These inputs would cause server errors anyway;
// rejecting them early provides clearer error messages.
func ResourceID(id string) error {
	if id == "" {
		return fmt.Errorf("resource ID must not be empty")
	}

	for _, r := range id {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("resource ID contains invalid control character")
		}
	}

	if strings.ContainsAny(id, "?&#") {
		return fmt.Errorf("resource ID must not contain query parameters (?, &, #)")
	}

	if strings.Contains(id, "../") || strings.Contains(id, "..\\") {
		return fmt.Errorf("resource ID must not contain path traversal sequences")
	}

	if strings.Contains(id, "%") {
		return fmt.Errorf("resource ID must not contain percent-encoded sequences")
	}

	return nil
}
