package guardrails

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// newBudgetRulesCreateCmd builds `revenium guardrails budget-rules create ...`.
//
// Per GRDR-03 / RESEARCH D-10 (verified against CostControlResource_Write.required),
// 8 fields are required: name, description, metricType, windowType, action, groupBy,
// warnThreshold, hardLimit. All 8 are enforced at the Cobra layer via
// MarkFlagRequired so under-specified invocations fail BEFORE any HTTP round-trip
// (T-14-03-05 mitigation).
//
// Optional fields shadowMode and enabled are gated by c.Flags().Changed(...) so
// they only appear in the request body when the user explicitly passed them
// (mirror of RESEARCH A3 gating pattern from cmd/jobs/create.go).
//
// PLAN 260524-kvj adds three more optional flags (also gated by Flags().Changed):
//   - --filter dim:op:val (repeatable) — composes filters array
//   - --filters-json '<JSON>'         — escape hatch (mutually exclusive with --filter, D-01)
//   - --notification-channel-id <id>  (repeatable) — composes notificationChannelIds array
//
// teamId and tenantId are auto-injected by cmd.APIClient.DoCreate from client
// config — NEVER exposed as CLI flags (CF-12-20).
func newBudgetRulesCreateCmd() *cobra.Command {
	var (
		name                   string
		description            string
		metricType             string
		windowType             string
		action                 string
		groupBy                string
		warnThreshold          float64
		hardLimit              float64
		shadowMode             bool
		enabled                bool
		filterFlags            []string
		filtersJSON            string
		notificationChannelIDs []string
	)

	c := &cobra.Command{
		Use:         "create",
		Short:       "Create a budget rule",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"mutating": "true"}, // CF-12-16
		Example: `  # Cap monthly OpenAI spend per model with a soft warn at 800 and hard block at 1000
  revenium guardrails budget-rules create \
    --name "Q3 OpenAI Budget" \
    --description "Caps monthly OpenAI spend" \
    --metric-type TOTAL_COST \
    --window-type MONTHLY \
    --action BLOCK \
    --group-by MODEL \
    --warn-threshold 800 \
    --hard-limit 1000

  # Same rule, scoped to a specific model with a notification channel attached
  revenium guardrails budget-rules create \
    --name "Q3 OpenAI Budget" \
    --description "Caps monthly OpenAI spend" \
    --metric-type TOTAL_COST \
    --window-type MONTHLY \
    --action BLOCK \
    --group-by MODEL \
    --warn-threshold 800 \
    --hard-limit 1000 \
    --filter MODEL:IS:gpt-4 \
    --notification-channel-id chan-123

  # Scope to models whose name contains "gpt" (string match)
  revenium guardrails budget-rules create \
    --name "GPT models budget" \
    --description "Caps spend on GPT family models" \
    --metric-type TOTAL_COST \
    --window-type MONTHLY \
    --action BLOCK \
    --group-by MODEL \
    --warn-threshold 800 \
    --hard-limit 1000 \
    --filter MODEL:CONTAINS:gpt

  # Same rule, but evaluate without blocking (shadow mode) and start disabled
  revenium guardrails budget-rules create \
    --name "Q3 OpenAI Budget (shadow)" \
    --description "Shadow-mode validation pass" \
    --metric-type TOTAL_COST \
    --window-type MONTHLY \
    --action BLOCK \
    --group-by MODEL \
    --warn-threshold 800 \
    --hard-limit 1000 \
    --shadow-mode \
    --enabled=false`,
		RunE: func(c *cobra.Command, args []string) error {
			// PLAN D-01: resolve filters FIRST so the mutual-exclusion check
			// between --filter and --filters-json fires before any HTTP call
			// (and before the dry-run gate, so dry-run also surfaces the
			// conflict instead of pretending to succeed).
			filters, filtersChanged, err := resolveFilters(c, filterFlags, filtersJSON)
			if err != nil {
				return err
			}

			// 8 OAS-required fields go in unconditionally — MarkFlagRequired below
			// ensures these variables are populated before RunE fires.
			body := map[string]interface{}{
				"name":          name,
				"description":   description,
				"metricType":    metricType,
				"windowType":    windowType,
				"action":        action,
				"groupBy":       groupBy,
				"warnThreshold": warnThreshold,
				"hardLimit":     hardLimit,
			}
			// Optional fields gated by Flags().Changed so unset flags never
			// leak default values into the request body (RESEARCH D-10 + PLAN A3).
			if c.Flags().Changed("shadow-mode") {
				body["shadowMode"] = shadowMode
			}
			if c.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			if filtersChanged {
				body["filters"] = filters
			}
			if c.Flags().Changed("notification-channel-id") {
				body["notificationChannelIds"] = notificationChannelIDs
			}

			if cmd.DryRun() { // CF-12-17
				return dryrun.Render(cmd.Output, "create", "budget rule", "/v2/api/ai/cost-controls", body)
			}

			var result map[string]interface{}
			// CF-12-20: DoCreate auto-injects teamId/tenantId from client config.
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/ai/cost-controls", body, &result); err != nil {
				return err
			}
			return renderRule(result)
		},
	}

	c.Flags().StringVar(&name, "name", "", "Display name for this rule")
	c.Flags().StringVar(&description, "description", "", "Human-readable description (pass \"\" for none)")
	c.Flags().StringVar(&metricType, "metric-type", "", "One of TOTAL_COST, TOKEN_COUNT, INPUT_TOKEN_COUNT, OUTPUT_TOKEN_COUNT, CACHED_TOKEN_COUNT, ERROR_COUNT, IMAGE_COUNT, VIDEO_COUNT, AUDIO_COUNT, CHARACTER_COUNT, CREDITS_CONSUMED")
	c.Flags().StringVar(&windowType, "window-type", "", "One of DAILY, WEEKLY, MONTHLY, QUARTERLY")
	c.Flags().StringVar(&action, "action", "BLOCK", "Enforcement action (currently only BLOCK)")
	c.Flags().StringVar(&groupBy, "group-by", "", "One of ORGANIZATION, CREDENTIAL, PRODUCT, MODEL, PROVIDER, AGENT, SUBSCRIBER, TASK_TYPE")
	c.Flags().Float64Var(&warnThreshold, "warn-threshold", 0, "Soft threshold that triggers a warning without blocking")
	c.Flags().Float64Var(&hardLimit, "hard-limit", 0, "Hard enforcement threshold")
	c.Flags().BoolVar(&shadowMode, "shadow-mode", false, "Evaluate without blocking (validation mode)")
	c.Flags().BoolVar(&enabled, "enabled", true, "Whether the rule is active")
	// PLAN 260524-kvj D-06: help text lists known dims/ops as a hint only; values
	// are passed through verbatim and the server validates them.
	c.Flags().StringArrayVar(&filterFlags, "filter", nil, "Repeatable filter in dim:op:val form (e.g. --filter MODEL:IS:gpt-4). Known dimensions: AGENT, MODEL, PROVIDER, ORGANIZATION, CREDENTIAL, PRODUCT, SUBSCRIBER, TASK_TYPE. Known operators: IS, IS_NOT, CONTAINS, STARTS_WITH, ENDS_WITH. Server validates values.")
	c.Flags().StringVar(&filtersJSON, "filters-json", "", "Alternative to --filter: full filters array as JSON, e.g. '[{\"dimension\":\"MODEL\",\"operator\":\"IS\",\"value\":\"gpt-4\"}]'. Mutually exclusive with --filter.")
	c.Flags().StringArrayVar(&notificationChannelIDs, "notification-channel-id", nil, "Repeatable notification channel ID to attach to this rule")

	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("description")
	_ = c.MarkFlagRequired("metric-type")
	_ = c.MarkFlagRequired("window-type")
	_ = c.MarkFlagRequired("action")
	_ = c.MarkFlagRequired("group-by")
	_ = c.MarkFlagRequired("warn-threshold")
	_ = c.MarkFlagRequired("hard-limit")

	return c
}
