// Package products implements the products CRUD commands for the Revenium CLI.
package products

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/output"
)

// Cmd is the parent products command, exported for registration in main.go.
var Cmd = &cobra.Command{
	Use:   "products",
	Short: "Manage products",
	Example: `  # List all products
  revenium products list

  # Get a specific product
  revenium products get prod-123

  # Create a product
  revenium products create --name "My Product"`,
}

func init() {
	Cmd.AddCommand(newListCmd())
	Cmd.AddCommand(newGetCmd())
	Cmd.AddCommand(newCreateCmd())
	Cmd.AddCommand(newUpdateCmd())
	Cmd.AddCommand(newDeleteCmd())
}

// tableDef defines the table layout for product output.
var tableDef = output.TableDef{
	Headers:      []string{"ID", "Name", "Status"},
	StatusColumn: 2,
}

// toRows converts a slice of product maps to table row strings.
func toRows(products []map[string]interface{}) [][]string {
	rows := make([][]string, len(products))
	for i, p := range products {
		rows[i] = []string{
			str(p, "id"),
			str(p, "name"),
			str(p, "status"),
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

// renderProduct renders a single product as a single-row table or JSON.
func renderProduct(product map[string]interface{}) error {
	rows := [][]string{{
		str(product, "id"),
		str(product, "name"),
		str(product, "status"),
	}}
	return cmd.Output.Render(tableDef, rows, product)
}
