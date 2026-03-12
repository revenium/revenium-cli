package models

import (
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// pricingCmd is the parent pricing subcommand under models.
var pricingCmd = &cobra.Command{
	Use:   "pricing",
	Short: "Manage pricing dimensions for AI models",
	Example: `  # List pricing dimensions for a model
  revenium models pricing list abc-123

  # Create a pricing dimension
  revenium models pricing create abc-123 --name "Input Tokens" --type input --price 0.003`,
}

// initPricing registers pricing subcommands. Called from models.go init() to avoid
// file-ordering issues with Go's init() functions.
func initPricing() {
	pricingCmd.AddCommand(newPricingListCmd())
	pricingCmd.AddCommand(newPricingCreateCmd())
	pricingCmd.AddCommand(newPricingUpdateCmd())
	pricingCmd.AddCommand(newPricingDeleteCmd())
}

// pricingTableDef defines the table layout for pricing dimension output.
var pricingTableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Type", "Price"},
	StatusColumn: -1,
}

// toPricingRows converts a slice of pricing dimension maps to table row strings.
func toPricingRows(dimensions []map[string]interface{}) [][]string {
	rows := make([][]string, len(dimensions))
	for i, d := range dimensions {
		rows[i] = []string{
			str(d, "id"),
			str(d, "name"),
			str(d, "dimensionType"),
			str(d, "unitPrice"),
		}
	}
	return rows
}

// renderPricingDimension renders a single pricing dimension as a single-row table or JSON.
func renderPricingDimension(dimension map[string]interface{}) error {
	rows := [][]string{{
		str(dimension, "id"),
		str(dimension, "name"),
		str(dimension, "dimensionType"),
		str(dimension, "unitPrice"),
	}}
	return cmd.Output.Render(pricingTableDef, rows, dimension)
}
