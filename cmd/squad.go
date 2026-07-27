package cmd

import (
	"github.com/spf13/cobra"
)

// SquadFlags holds the squad identity flag values bound to a command by
// AddSquadFlags. Zero values mean the flag was not passed; use
// c.Flags().Changed(...) (via ApplySquadFlags) to distinguish "not passed"
// from "passed as empty string".
type SquadFlags struct {
	ID   string
	Name string
	Role string
}

// AddSquadFlags registers --squad-id, --squad-name, and --squad-role on c,
// binding their values into v. Call ApplySquadFlags after building the
// request body to inject the squad fields only for flags the operator
// explicitly passed.
func AddSquadFlags(c *cobra.Command, v *SquadFlags) {
	c.Flags().StringVar(&v.ID, "squad-id", "", "Squad identifier")
	c.Flags().StringVar(&v.Name, "squad-name", "", "Squad name")
	c.Flags().StringVar(&v.Role, "squad-role", "", "Agent's role within the squad (free-form, e.g. planner, executor, reviewer)")
}

// ApplySquadFlags injects squadId, squadName, and squadRole into body, but
// only for the flags that were explicitly passed on c (per-flag
// c.Flags().Changed(...) gate). Flags left at their default are omitted
// from body entirely rather than sent as empty strings.
func ApplySquadFlags(c *cobra.Command, body map[string]interface{}, v SquadFlags) {
	if c.Flags().Changed("squad-id") {
		body["squadId"] = v.ID
	}
	if c.Flags().Changed("squad-name") {
		body["squadName"] = v.Name
	}
	if c.Flags().Changed("squad-role") {
		body["squadRole"] = v.Role
	}
}
