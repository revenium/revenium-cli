// Package main is the entry point for the Revenium CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/revenium/revenium-cli/cmd"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if cmd.JSONMode() {
			// In JSON mode, errors go to stderr as JSON
			f := output.New(true, false)
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				f.RenderJSONError(apiErr.Message, apiErr.StatusCode)
			} else {
				f.RenderJSONError(err.Error(), 0)
			}
		} else {
			fmt.Fprintln(os.Stderr, apierrors.RenderError(err.Error()))
		}
		os.Exit(1)
	}
}
