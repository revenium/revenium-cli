package metrics

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

var completionsTableDef = output.TableDef{
	Headers:      []string{"ID", "Model", "Input", "Output", "Cached", "Reasoning", "TTFT", "Tok/Min", "Duration", "Stop Reason", "Cost", "Organization", "Agent", "Subscriber", "Squad"},
	StatusColumn: -1,
}

// promptsTableDef and referenceDataTableDef render single-object responses
// as 2-column key/value tables (mirrors cmd/jobs/roi.go's renderROI pattern)
// since GET .../completions/{id}/prompts and .../completions/reference-data
// both return a single flat object (PromptDataResource /
// AICompletionMetricReferenceData per the live platform spec), not a list —
// deviates from 04-05-PLAN.md's "prompts is likely a list (DoList)"
// assumption, which the live spec disproves (Rule 1: fix to match the
// verified source of truth).
var promptsTableDef = output.TableDef{
	Headers:      []string{"Field", "Value"},
	StatusColumn: -1,
}

var referenceDataTableDef = output.TableDef{
	Headers:      []string{"Field", "Value"},
	StatusColumn: -1,
}

func newCompletionsCmd() *cobra.Command {
	var squadID string
	c := &cobra.Command{
		Use:   "completions",
		Short: "Query AI completion metrics",
		Args:  cobra.NoArgs,
		Example: `  # Query completion metrics for last 24 hours
  revenium metrics completions

  # Query with time range
  revenium metrics completions --from 2024-01-01T00:00:00Z --to 2024-01-31T23:59:59Z

  # Filter to a single squad (client-side)
  revenium metrics completions --squad-id squad-loan-proc-12345`,
		RunE: func(c *cobra.Command, args []string) error {
			var metrics []map[string]interface{}
			path := buildPath("/v2/api/sources/metrics/ai/completions")
			opts := cmd.ListOptsFromFlags(c)
			if c.Flags().Changed("squad-id") {
				// D-10/Pitfall 5: force a full-pageset fetch so a match on a
				// later page isn't missed. FetchAll alone is not enough —
				// api.Client.DoList only fetches all pages when neither Page
				// nor PageSize is explicitly set, so any explicit --page/
				// --page-size must also be cleared here.
				opts.FetchAll = true
				opts.Page = -1
				opts.PageSize = -1
			}
			if err := cmd.APIClient.DoList(c.Context(), path, opts, &metrics); err != nil {
				return err
			}
			if c.Flags().Changed("squad-id") {
				// Client-side exact-match filter (D-09) — no server-side
				// squadId query param exists on this endpoint.
				metrics = filterBySquadID(metrics, squadID)
			}
			if len(metrics) == 0 {
				if cmd.Output.IsJSON() {
					return cmd.Output.RenderJSON([]interface{}{})
				}
				fmt.Fprintln(c.OutOrStdout(), "No metrics found.")
				return nil
			}
			return cmd.Output.Render(completionsTableDef, toCompletionRows(metrics), metrics)
		},
	}

	c.Flags().StringVar(&squadID, "squad-id", "", "Filter to a single squad (client-side, exact match on squadId — no server-side squad filter exists on this endpoint)")
	cmd.AddListFlags(c)

	// MTR-06 platform-host completion-detail lookups (D-05: extend this
	// existing `metrics completions` group rather than a new top-level
	// group). All three stay on the existing platform base URL/x-api-key
	// auth — no bearer-auth flag flip, no analytics base.
	c.AddCommand(newCompletionsGetCmd(), newCompletionsPromptsCmd(), newCompletionsReferenceDataCmd())
	return c
}

// newCompletionsGetCmd returns `revenium metrics completions get <id>`.
// Endpoint: GET /v2/api/sources/metrics/ai/completions/{url.PathEscape(id)}
// (platform host). Response is an AICompletionMetricResource — the same
// shape as each row of the list endpoint above, so it is rendered through
// the existing completionsTableDef/toCompletionRows for consistency.
func newCompletionsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single AI completion metric by ID",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get a completion metric by ID
  revenium metrics completions get txn-abc123

  # Get a completion metric as JSON
  revenium metrics completions get txn-abc123 --json`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/sources/metrics/ai/completions/%s", url.PathEscape(args[0]))
			var m map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &m); err != nil {
				return err
			}
			return cmd.Output.Render(completionsTableDef, toCompletionRows([]map[string]interface{}{m}), m)
		},
	}
}

// newCompletionsPromptsCmd returns `revenium metrics completions prompts <id>`.
// Endpoint: GET /v2/api/sources/metrics/ai/completions/{id}/prompts (platform
// host). Response is a single PromptDataResource object (systemPrompt,
// inputMessages, outputResponse, promptsTruncated) per the live platform
// spec — NOT a list, so this uses single-object Do (not DoList).
func newCompletionsPromptsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prompts <id>",
		Short: "Get prompt data for an AI completion",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Get prompt data for a completion
  revenium metrics completions prompts txn-abc123`,
		RunE: func(c *cobra.Command, args []string) error {
			path := fmt.Sprintf("/v2/api/sources/metrics/ai/completions/%s/prompts", url.PathEscape(args[0]))
			var p map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", path, nil, &p); err != nil {
				return err
			}
			rows := [][]string{
				{"System Prompt", str(p, "systemPrompt")},
				{"Input Messages", str(p, "inputMessages")},
				{"Output Response", str(p, "outputResponse")},
				{"Prompts Truncated", str(p, "promptsTruncated")},
			}
			return cmd.Output.Render(promptsTableDef, rows, p)
		},
	}
}

// newCompletionsReferenceDataCmd returns `revenium metrics completions reference-data`.
// Endpoint: GET /v2/api/sources/metrics/ai/completions/reference-data
// (platform host, no id/args). Response is a single AICompletionMetricReferenceData
// object with two string-array fields (traceTypes, stopReasons) — rendered as
// a 2-row key/value table with each array comma-joined.
func newCompletionsReferenceDataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reference-data",
		Short: "Get reference data for AI completion metric search filters",
		Args:  cobra.NoArgs,
		Example: `  # Get completion search-filter reference data (trace types, stop reasons)
  revenium metrics completions reference-data`,
		RunE: func(c *cobra.Command, args []string) error {
			var r map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "GET", "/v2/api/sources/metrics/ai/completions/reference-data", nil, &r); err != nil {
				return err
			}
			rows := [][]string{
				{"Trace Types", joinStrList(r, "traceTypes")},
				{"Stop Reasons", joinStrList(r, "stopReasons")},
			}
			return cmd.Output.Render(referenceDataTableDef, rows, r)
		},
	}
}

// joinStrList extracts a []interface{} of strings from a map and comma-joins
// them for display. Returns "" for a missing/empty/non-array key.
func joinStrList(m map[string]interface{}, key string) string {
	raw, ok := m[key].([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(raw))
	for _, v := range raw {
		if v != nil {
			parts = append(parts, fmt.Sprint(v))
		}
	}
	return strings.Join(parts, ", ")
}

func toCompletionRows(metrics []map[string]interface{}) [][]string {
	rows := make([][]string, len(metrics))
	for i, m := range metrics {
		rows[i] = []string{
			str(m, "transactionId"),
			str(m, "model"),
			formatNumber(floatVal(m, "inputTokenCount")),
			formatNumber(floatVal(m, "outputTokenCount")),
			formatNumber(floatVal(m, "cacheReadTokenCount")),
			formatNumber(floatVal(m, "reasoningTokenCount")),
			formatDuration(floatVal(m, "timeToFirstToken")),
			formatNumber(floatVal(m, "tokensPerMinute")),
			formatDuration(floatVal(m, "requestDuration")),
			str(m, "stopReason"),
			formatCost(floatVal(m, "totalCost")),
			nestedStr(m, "organization", "label"),
			str(m, "agent"),
			nestedStr(m, "subscriberCredential", "label"),
			squadCell(m),
		}
	}
	return rows
}
