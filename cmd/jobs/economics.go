package jobs

import (
	"github.com/spf13/cobra"
)

// economicsCmd is the parent `economics` sub-resource under `jobs types`
// (D-27-07).
//
// This file declares NO package initializer. Go runs package initializers in
// lexical filename order, and economics.go sorts before types.go, so an
// initializer here would run first and see a nil typesCmd — registering
// nothing, or panicking. Registration therefore lives in types.go and calls
// initEconomics() below, mirroring cmd/teams/prompt_capture.go.
var economicsCmd = &cobra.Command{
	Use:   "economics",
	Short: "Manage a job type's declared economics contract",
	Example: `  # Read a job type's declared economics contract
  revenium jobs types economics get acme-review

  # As JSON for scripting — the server resource passes through untouched
  revenium jobs types economics get acme-review --json`,
}

// initEconomics registers the economics subcommands. Called from types.go's
// package initializer after typesCmd is assigned — see the comment on
// economicsCmd for why it is not called from this file.
func initEconomics() {
	economicsCmd.AddCommand(newEconomicsGetCmd())
	economicsCmd.AddCommand(newEconomicsSetCmd())
}
