package cmd

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/validate"
)

// ValidResourceID is a Cobra-compatible Args validator that checks all
// positional arguments are valid resource IDs (no control chars, query
// params, path traversal, or percent-encoding).
func ValidResourceID(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if err := validate.ResourceID(arg); err != nil {
			return err
		}
	}
	return nil
}
