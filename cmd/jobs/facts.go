package jobs

import (
	"github.com/spf13/cobra"
)

// factsCmd is the parent `facts` sub-resource under `jobs types` (JOBS-15).
//
// This file declares NO package initializer. Go runs package initializers in
// lexical filename order, and facts.go sorts before types.go, so an
// initializer here would run first and see a nil typesCmd — registering
// nothing, or panicking on every `revenium` invocation including --help.
// Registration therefore lives in types.go, mirroring baselines.go and
// economics.go.
//
// The help text below deliberately never says an appended fact can be changed,
// nor that one can be read back afterwards. PeriodFactEntry is append-only:
// the API declares no endpoint that amends a fact and none that returns one,
// which is why `facts` has no `list` sibling where `baselines` has one. An
// operator who read otherwise here would form a belief the platform will not
// honour.
var factsCmd = &cobra.Command{
	Use:   "facts",
	Short: "Append a job type's declared period outcome facts",
	Long: "Append a job type's declared period outcome facts.\n\n" +
		"Each entry states one measured value for one metric over one period.\n" +
		"Entries are permanent and are sent in the order the input supplies\n" +
		"them; the platform offers no way to withdraw one afterwards, and no\n" +
		"way to read one back, so this command reports what it sent. Preview a\n" +
		"file with --dry-run when in doubt.",
	Example: `  # Append the period facts held in a file
  revenium jobs types facts append acme-review --file facts.json

  # Append from a pipe
  cat facts.json | revenium jobs types facts append acme-review --file -

  # Preview what would be sent, sending nothing
  revenium jobs types facts append acme-review --file facts.json --dry-run`,
}

// initFacts registers the facts subcommands. Called from types.go's package
// initializer after typesCmd is assigned — see the comment on factsCmd for why
// it is not called from this file.
func initFacts() {
	factsCmd.AddCommand(newFactsAppendCmd())
}
