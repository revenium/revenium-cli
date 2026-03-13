package cmd

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/api"
)

// AddListFlags adds --page and --page-size flags to a list command.
func AddListFlags(c *cobra.Command) {
	c.Flags().Int("page", 0, "Page number (0-based)")
	c.Flags().Int("page-size", 20, "Number of items per page")
}

// ListOptsFromFlags builds ListOptions from the command's flags.
// In table mode (non-JSON), FetchAll defaults to true so all pages are aggregated.
// In JSON mode, single-page API behavior is used by default.
// Explicitly setting --page or --page-size always uses single-page mode.
func ListOptsFromFlags(c *cobra.Command) api.ListOptions {
	explicitPaging := c.Flags().Changed("page") || c.Flags().Changed("page-size")

	page := -1
	pageSize := -1
	if explicitPaging {
		page, _ = c.Flags().GetInt("page")
		pageSize, _ = c.Flags().GetInt("page-size")
		if !c.Flags().Changed("page") {
			page = -1
		}
		if !c.Flags().Changed("page-size") {
			pageSize = -1
		}
	}

	return api.ListOptions{
		Page:     page,
		PageSize: pageSize,
		FetchAll: !Output.IsJSON() && !explicitPaging,
	}
}
