package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apierrors "github.com/revenium/revenium-cli/internal/errors"
)

func newSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Output the full CLI command tree as machine-readable JSON",
		Long:  "Dumps the complete command structure, flags, and metadata as JSON for programmatic discovery by AI agents and scripts.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			root := Root()
			schema := buildSchema(root)
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(schema)
		},
	}
}

func init() {
	schemaCmd := newSchemaCmd()
	schemaCmd.GroupID = "config"
	rootCmd.AddCommand(schemaCmd)
}

type schemaOutput struct {
	Commands    []commandSchema `json:"commands"`
	GlobalFlags []flagSchema    `json:"global_flags"`
	ExitCodes   map[string]int  `json:"exit_codes"`
}

type commandSchema struct {
	Path        string          `json:"path"`
	Description string          `json:"description"`
	Flags       []flagSchema    `json:"flags,omitempty"`
	Args        string          `json:"args,omitempty"`
	Mutating    bool            `json:"mutating,omitempty"`
	Subcommands []commandSchema `json:"subcommands,omitempty"`
}

type flagSchema struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage"`
	Required  bool   `json:"required,omitempty"`
}

func buildSchema(root *cobra.Command) schemaOutput {
	return schemaOutput{
		Commands:    walkCommands(root, ""),
		GlobalFlags: extractFlags(root.PersistentFlags()),
		ExitCodes: map[string]int{
			"ok":         apierrors.ExitOK,
			"general":    apierrors.ExitGeneral,
			"auth":       apierrors.ExitAuth,
			"not_found":  apierrors.ExitNotFound,
			"validation": apierrors.ExitValidation,
			"network":    apierrors.ExitNetwork,
		},
	}
}

func walkCommands(c *cobra.Command, parentPath string) []commandSchema {
	var commands []commandSchema
	for _, sub := range c.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" || sub.Name() == "schema" {
			continue
		}

		path := sub.Name()
		if parentPath != "" {
			path = parentPath + " " + sub.Name()
		}

		cs := commandSchema{
			Path:        path,
			Description: sub.Short,
		}

		// Check for mutating annotation
		if sub.Annotations != nil {
			if _, ok := sub.Annotations["mutating"]; ok {
				cs.Mutating = true
			}
		}

		// Extract args spec from Use string
		if parts := strings.SplitN(sub.Use, " ", 2); len(parts) > 1 {
			cs.Args = parts[1]
		}

		// Local flags (not inherited)
		cs.Flags = extractLocalFlags(sub)

		// Recurse into subcommands
		children := walkCommands(sub, path)
		if len(children) > 0 {
			cs.Subcommands = children
		}

		commands = append(commands, cs)
	}
	return commands
}

func extractFlags(flags *pflag.FlagSet) []flagSchema {
	var result []flagSchema
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		fs := flagSchema{
			Name:  f.Name,
			Type:  f.Value.Type(),
			Usage: f.Usage,
		}
		if f.Shorthand != "" {
			fs.Shorthand = f.Shorthand
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			fs.Default = f.DefValue
		}
		result = append(result, fs)
	})
	return result
}

func extractLocalFlags(c *cobra.Command) []flagSchema {
	requiredFlags := make(map[string]bool)
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if ann := f.Annotations; ann != nil {
			if _, ok := ann[cobra.BashCompOneRequiredFlag]; ok {
				requiredFlags[f.Name] = true
			}
		}
	})

	var result []flagSchema
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		// Skip inherited persistent flags
		if c.Parent() != nil && c.Parent().PersistentFlags().Lookup(f.Name) != nil {
			return
		}
		fs := flagSchema{
			Name:     f.Name,
			Type:     f.Value.Type(),
			Usage:    f.Usage,
			Required: requiredFlags[f.Name],
		}
		if f.Shorthand != "" {
			fs.Shorthand = f.Shorthand
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			fs.Default = f.DefValue
		}
		result = append(result, fs)
	})
	return result
}
