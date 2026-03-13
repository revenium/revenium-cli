// Package dryrun provides dry-run rendering for mutation commands.
package dryrun

import (
	"fmt"

	"github.com/revenium/revenium-cli/internal/output"
)

// Render outputs a dry-run summary without executing the API call.
// In JSON mode it emits a structured object; in table mode a human-readable summary.
func Render(f *output.Formatter, action, resource, path string, body interface{}) error {
	if f.IsJSON() {
		return f.RenderJSON(map[string]interface{}{
			"dry_run":  true,
			"action":   action,
			"resource": resource,
			"path":     path,
			"body":     body,
		})
	}

	fmt.Fprintf(f.Writer(), "Dry run: %s %s\n", action, resource)
	fmt.Fprintf(f.Writer(), "  Path: %s\n", path)
	if body != nil {
		fmt.Fprintf(f.Writer(), "  Body: %v\n", body)
	}
	fmt.Fprintf(f.Writer(), "\nNo changes were made.\n")
	return nil
}
