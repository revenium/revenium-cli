// Package main is the entry point for the Revenium CLI.
package main

import (
	"fmt"
	"os"

	"github.com/revenium/revenium-cli/cmd"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, apierrors.RenderError(err.Error()))
		os.Exit(1)
	}
}
