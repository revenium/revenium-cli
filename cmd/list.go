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
//
// All pages are aggregated by default, in every output mode. Only an explicit
// --page opts out, because only --page expresses "give me exactly this one
// page"; --page-size merely selects the batch size to fetch in.
//
// This deliberately does NOT special-case JSON mode. JSON is the automation
// path, where silently returning the first page — while the caller believes it
// received the whole result set — is the most damaging possible default. A
// month-of-spend query that quietly returns only the newest page is
// indistinguishable from an accurate one until the numbers are reconciled
// against another source.
func ListOptsFromFlags(c *cobra.Command) api.ListOptions {
	page := -1
	pageSize := -1
	if c.Flags().Changed("page") {
		page, _ = c.Flags().GetInt("page")
	}
	if c.Flags().Changed("page-size") {
		pageSize, _ = c.Flags().GetInt("page-size")
	}

	return api.ListOptions{
		Page:     page,
		PageSize: pageSize,
		FetchAll: page < 0,
	}
}
