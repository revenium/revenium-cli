package models

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
)

func newPricingCreateCmd() *cobra.Command {
	var (
		billingUnit      string
		modality         string
		costType         string
		direction        string
		operationSubtype string
		price            float64
	)

	c := &cobra.Command{
		Use:   "create <model-id>",
		Short:       "Create a pricing dimension for a model",
		Annotations: map[string]string{"mutating": "true"},
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Create a text input token pricing dimension
  revenium models pricing create model-123 --billing-unit PER_TOKEN --modality TEXT --cost-type TEXT_TOKEN_INPUT --direction INPUT --price 0.003

  # Create an image generation pricing dimension
  revenium models pricing create model-123 --billing-unit PER_IMAGE --modality IMAGE --cost-type IMAGE_GENERATION --price 0.04`,
		RunE: func(c *cobra.Command, args []string) error {
			modelID := args[0]
			path := fmt.Sprintf("/v2/api/sources/ai/models/%s/pricing/dimensions", modelID)

			body := map[string]interface{}{
				"billingUnit": billingUnit,
				"modality":    modality,
				"costType":    costType,
				"unitPrice":   price,
			}
			if c.Flags().Changed("direction") {
				body["direction"] = direction
			}
			if c.Flags().Changed("operation-subtype") {
				body["operationSubtype"] = operationSubtype
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "create", "pricing dimension", path, body)
			}

			var result map[string]interface{}
			if err := cmd.APIClient.Do(c.Context(), "POST", path, body, &result); err != nil {
				return err
			}
			return renderPricingDimension(result)
		},
	}

	c.Flags().StringVar(&billingUnit, "billing-unit", "", "Billing unit (PER_TOKEN, PER_IMAGE, PER_SECOND, PER_MINUTE, PER_CHARACTER, CREDITS)")
	c.Flags().StringVar(&modality, "modality", "", "Modality (TEXT, IMAGE, AUDIO, VIDEO)")
	c.Flags().StringVar(&costType, "cost-type", "", "Cost type (TEXT_TOKEN_INPUT, TEXT_TOKEN_OUTPUT, IMAGE_GENERATION, etc.)")
	c.Flags().StringVar(&direction, "direction", "", "Direction (INPUT, OUTPUT, BIDIRECTIONAL)")
	c.Flags().StringVar(&operationSubtype, "operation-subtype", "", "Operation subtype (e.g., input, output, generation)")
	c.Flags().Float64Var(&price, "price", 0, "Unit price in USD")
	_ = c.MarkFlagRequired("billing-unit")
	_ = c.MarkFlagRequired("modality")
	_ = c.MarkFlagRequired("cost-type")

	return c
}
