package teams

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

// verifiedDomainsCmd is the parent verified-domains subcommand under teams.
//
// D-31-10 — why this group is `list | add | remove` and NOT the house
// `get | set | delete` settings pair.
//
// The sibling settings groups in this package — `prompt-capture`,
// `coding-assistant-filter`, and this phase's two new scalar groups
// `pr-health` and `attribution-identity-policy` — are all `get|set` because
// their endpoints are SCALAR SETTINGS: one PUT replaces the whole thing, so
// `set` describes the effect exactly.
//
// This endpoint is a COLLECTION wearing a settings URL. Its GET returns an
// array, its PUT APPENDS one member, and its DELETE removes one member matched
// by a query parameter. Naming an append `set` would tell an operator the
// opposite of what happens — that the call replaces the collection.
//
// The standing rule (D-27-07) is that command names mirror the URL. Where the
// URL's method and its effect disagree, the operator-facing name follows the
// EFFECT. The asymmetry between this group and its `get|set` siblings is
// therefore deliberate, not an oversight; do not "harmonise" it.
var verifiedDomainsCmd = &cobra.Command{
	Use:   "verified-domains",
	Short: "List, add and remove verified domains for a team's tenant",
	Example: `  # List the tenant's verified domains
  revenium teams verified-domains list team-123

  # Verify a domain for the tenant
  revenium teams verified-domains add team-123 acme.example

  # Remove a verified domain (with confirmation)
  revenium teams verified-domains remove team-123 acme.example`,
}

// initVerifiedDomains registers verified-domains subcommands. Called from
// teams.go init() to avoid file-ordering issues with Go's init() functions
// (mirrors initPromptCapture / initCodingAssistantFilter).
func initVerifiedDomains() {
	verifiedDomainsCmd.AddCommand(newVerifiedDomainsListCmd())
	verifiedDomainsCmd.AddCommand(newVerifiedDomainsAddCmd())
	verifiedDomainsCmd.AddCommand(newVerifiedDomainsRemoveCmd())
}

// verifiedDomainsTableDef defines the table layout for verified-domain rows,
// shared by `list` and `add`.
//
// Three columns, not one. `source` and `joinPolicy` are server-assigned — the
// operator cannot set either (VerifiedDomainRequest declares a single property)
// — which is precisely why they are rendered rather than dropped: an operator
// who verifies a domain should see the join policy the platform attached to it
// (D-31-13). `add` renders the single row it just created through this same
// definition, so the collection view and the confirmation view agree.
var verifiedDomainsTableDef = output.TableDef{
	Headers:      []string{"Domain", "Source", "Join Policy"},
	StatusColumn: -1,
}

// verifiedDomainRows converts decoded VerifiedDomainResource objects to table
// rows, reusing the package's existing str() helper (teams.go) rather than
// redeclaring one.
func verifiedDomainRows(items []map[string]interface{}) [][]string {
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = []string{
			str(it, "domain"),
			str(it, "source"),
			str(it, "joinPolicy"),
		}
	}
	return rows
}
