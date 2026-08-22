package cmd

import (
	"github.com/spf13/cobra"
)

// AddTicketFlag registers --ticket-id on c, binding its value into v.
// The zero value means the flag was not passed; use ApplyTicketFlag to
// distinguish "not passed" from "passed as empty string".
//
// The field is shared by the four AI metering commands (completion, audio,
// image, video). It is absent from the tool-event schema, so meter
// tool-event must not call this.
func AddTicketFlag(c *cobra.Command, v *string) {
	c.Flags().StringVar(v, "ticket-id", "", "External ticket or issue ID for cost attribution (e.g. JIRA-123, LINEAR-456; max 256 characters)")
}

// ApplyTicketFlag injects ticketId into body, but only when --ticket-id was
// explicitly passed on c. Left at its default, the key is omitted from body
// entirely rather than sent as an empty string.
func ApplyTicketFlag(c *cobra.Command, body map[string]interface{}, v string) {
	if c.Flags().Changed("ticket-id") {
		body["ticketId"] = v
	}
}
