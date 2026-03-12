// Package main is the entry point for the Revenium CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/cmd/alerts"
	"github.com/revenium/revenium-cli/cmd/anomalies"
	"github.com/revenium/revenium-cli/cmd/charts"
	"github.com/revenium/revenium-cli/cmd/credentials"
	"github.com/revenium/revenium-cli/cmd/metrics"
	"github.com/revenium/revenium-cli/cmd/models"
	"github.com/revenium/revenium-cli/cmd/products"
	"github.com/revenium/revenium-cli/cmd/sources"
	"github.com/revenium/revenium-cli/cmd/subscribers"
	"github.com/revenium/revenium-cli/cmd/subscriptions"
	"github.com/revenium/revenium-cli/cmd/teams"
	"github.com/revenium/revenium-cli/cmd/tools"
	"github.com/revenium/revenium-cli/cmd/users"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
)

func init() {
	// Register resource commands here to avoid circular imports.
	// Resource packages (cmd/sources, etc.) import cmd for APIClient/Output,
	// so cmd/root.go cannot import them directly.
	cmd.RegisterCommand(sources.Cmd, "resources")
	cmd.RegisterCommand(models.Cmd, "resources")
	cmd.RegisterCommand(subscribers.Cmd, "resources")
	cmd.RegisterCommand(subscriptions.Cmd, "resources")
	cmd.RegisterCommand(products.Cmd, "resources")
	cmd.RegisterCommand(tools.Cmd, "resources")
	cmd.RegisterCommand(teams.Cmd, "resources")
	cmd.RegisterCommand(users.Cmd, "resources")
	cmd.RegisterCommand(anomalies.Cmd, "resources")
	cmd.RegisterCommand(alerts.Cmd, "resources")
	cmd.RegisterCommand(credentials.Cmd, "resources")
	cmd.RegisterCommand(charts.Cmd, "resources")
	cmd.RegisterCommand(metrics.Cmd, "monitoring")
}

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
