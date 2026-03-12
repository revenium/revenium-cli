// Package resource provides shared helpers for CRUD resource commands.
package resource

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

// ConfirmDelete prompts the user to confirm deletion of a resource.
// Returns true if the user confirms, or if skipConfirm is true,
// jsonMode is true, or stdin is not a TTY.
func ConfirmDelete(resourceType, id string, skipConfirm, jsonMode bool) (bool, error) {
	if skipConfirm || jsonMode {
		return true, nil
	}
	if !term.IsTerminal(os.Stdin.Fd()) {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "Delete %s %s? [y/N] ", resourceType, id)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
