package alerts

import "github.com/spf13/cobra"

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage budget alert thresholds",
}

func initBudget() {}
