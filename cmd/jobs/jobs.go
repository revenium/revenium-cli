// Package jobs implements the Agentic Jobs CRUD commands for the Revenium CLI.
package jobs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent jobs command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage Agentic Jobs",
	Example: `  # List all jobs
  revenium jobs list

  # Get a specific job by its agenticJobId
  revenium jobs get loan-app-12345

  # Create a new job
  revenium jobs create --agentic-job-id loan-app-12345 --name "Process Loan"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
}

// tableDef defines the table layout for job output (D-06/D-07).
// The third column is sourced from executionStatus per D-11 Option A — there
// is no top-level `status` field on JobResource_Read. Empty cell when
// executionStatus is absent (job still in flight) is intentional.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
}

// toRows converts a slice of job maps to table row strings.
// Column 2 is extracted from `executionStatus` (NOT `status`) per D-11 Option A.
func toRows(jobs []map[string]interface{}) [][]string {
	rows := make([][]string, len(jobs))
	for i, j := range jobs {
		rows[i] = []string{
			str(j, "id"),
			str(j, "label"),
			str(j, "executionStatus"),
		}
	}
	return rows
}

// str safely extracts a string value from a map, returning "" for missing or nil keys.
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// renderJob renders a single job as a single-row table or JSON.
// Used by get/create/update in this and subsequent plans.
func renderJob(job map[string]interface{}) error {
	rows := [][]string{{
		str(job, "id"),
		str(job, "label"),
		str(job, "executionStatus"),
	}}
	return cmd.Output.Render(tableDef, rows, job)
}
