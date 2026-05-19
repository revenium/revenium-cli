package jobs

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

// init registers newCreateCmd() onto the package-level jobs.Cmd. Per D-21 and
// Go's multi-init-per-package semantics, this composes with jobs.go's init()
// (which wires list+get). Phase 13 will add additional init() funcs in sibling
// files (outcome.go, roi.go, ...) the same way.
func init() {
	Cmd.AddCommand(newCreateCmd())
}

// newCreateCmd builds `revenium jobs create --agentic-job-id <id> [...]`.
//
// Per CONTEXT D-04 (resolved via OAS CreateJobRequest_Write.required), only
// `agenticJobId` is required. The other four fields (name, type, version,
// environment) are optional and gated by c.Flags().Changed(...) so empty
// strings never end up in the POST body (RESEARCH §"Required Create Fields" A3).
func newCreateCmd() *cobra.Command {
	var (
		agenticJobID string
		name         string
		jobType      string // Go identifier; the CLI flag is --type (a Go keyword)
		version      string
		environment  string
	)

	c := &cobra.Command{
		Use:         "create",
		Short:       "Create a new job",
		Annotations: map[string]string{"mutating": "true"}, // D-16
		Example: `  # Create a job with just the required identifier
  revenium jobs create --agentic-job-id loan-app-12345

  # Create with full metadata
  revenium jobs create --agentic-job-id loan-app-12345 --name "Process Loan" --type loan-processing --environment production`,
		RunE: func(c *cobra.Command, args []string) error {
			// agenticJobId is unconditionally included — it's the only OAS-required field (D-04).
			body := map[string]interface{}{
				"agenticJobId": agenticJobID,
			}
			// Gate optional fields per RESEARCH A3 so the server never sees an empty
			// string the caller didn't intend to send.
			if c.Flags().Changed("name") {
				body["name"] = name
			}
			if c.Flags().Changed("type") {
				body["type"] = jobType
			}
			if c.Flags().Changed("version") {
				body["version"] = version
			}
			if c.Flags().Changed("environment") {
				body["environment"] = environment
			}

			if cmd.DryRun() { // D-17
				return dryrun.Render(cmd.Output, "create", "job", "/v2/api/jobs", body)
			}

			var result map[string]interface{}
			// D-20: DoCreate auto-injects teamId/tenantId — never inline-concat ?teamId=.
			if err := cmd.APIClient.DoCreate(c.Context(), "/v2/api/jobs", body, &result); err != nil {
				return err
			}
			return renderJob(result) // reuse Plan 01 helper from jobs.go
		},
	}

	// D-05: long-flag-only convention for resource fields.
	c.Flags().StringVar(&agenticJobID, "agentic-job-id", "", "User-supplied external identifier (required)")
	c.Flags().StringVar(&name, "name", "", "Human-readable job name")
	c.Flags().StringVar(&jobType, "type", "", "Job category (e.g. loan-processing)")
	c.Flags().StringVar(&version, "version", "", "Job version identifier")
	c.Flags().StringVar(&environment, "environment", "", "Deployment environment (e.g. production)")
	// D-04: ONLY --agentic-job-id is required per OAS. Do NOT MarkFlagRequired
	// on name/type/version/environment (anomalies analog requires --name; jobs
	// deviates intentionally because the OAS does not).
	_ = c.MarkFlagRequired("agentic-job-id")

	return c
}
