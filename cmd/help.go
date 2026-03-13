package cmd

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// Help template styles
	helpHeading  = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	helpCommand  = lipgloss.NewStyle().Foreground(lipgloss.Color("73"))
	helpFlag     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpDesc     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpExample  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpComment  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpLong     = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	helpUsage    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	helpRequired = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

// SetupHelp configures colorized help output on the given command.
func SetupHelp(root *cobra.Command) {
	root.SetHelpFunc(func(c *cobra.Command, _ []string) {
		colorUsage(c)
	})
}

// colorUsage renders a fully colorized usage/help page for the command.
func colorUsage(c *cobra.Command) error {
	var b strings.Builder

	// Long or short description
	if c.Long != "" {
		b.WriteString(helpLong.Render(c.Long))
		b.WriteString("\n\n")
	} else if c.Short != "" {
		b.WriteString(helpLong.Render(c.Short))
		b.WriteString("\n\n")
	}

	// Usage
	b.WriteString(helpHeading.Render("Usage:"))
	b.WriteString("\n")
	if c.Runnable() {
		b.WriteString("  ")
		b.WriteString(helpUsage.Render(c.UseLine()))
		b.WriteString("\n")
	}
	if c.HasAvailableSubCommands() {
		b.WriteString("  ")
		b.WriteString(helpUsage.Render(c.CommandPath() + " [command]"))
		b.WriteString("\n")
	}

	// Examples
	if c.HasExample() {
		b.WriteString("\n")
		b.WriteString(helpHeading.Render("Examples:"))
		b.WriteString("\n")
		for _, line := range strings.Split(c.Example, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				b.WriteString(helpComment.Render(line))
			} else if trimmed != "" {
				b.WriteString(helpExample.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Grouped subcommands
	if groups := c.Groups(); len(groups) > 0 {
		for _, g := range groups {
			cmds := commandsInGroup(c, g.ID)
			if len(cmds) == 0 {
				continue
			}
			b.WriteString("\n")
			b.WriteString(helpHeading.Render(g.Title))
			b.WriteString("\n")
			writeCommandList(&b, cmds)
		}
		// Ungrouped commands
		ungrouped := ungroupedCommands(c)
		if len(ungrouped) > 0 {
			b.WriteString("\n")
			b.WriteString(helpHeading.Render("Additional Commands:"))
			b.WriteString("\n")
			writeCommandList(&b, ungrouped)
		}
	} else if c.HasAvailableSubCommands() {
		b.WriteString("\n")
		b.WriteString(helpHeading.Render("Available Commands:"))
		b.WriteString("\n")
		writeCommandList(&b, availableCommands(c))
	}

	// Flags
	localFlags := c.LocalFlags()
	inheritedFlags := c.InheritedFlags()

	if c.HasParent() {
		// Subcommand: show local flags + inherited (global) flags separately
		if hasVisibleFlags(localFlags) {
			b.WriteString("\n")
			b.WriteString(helpHeading.Render("Flags:"))
			b.WriteString("\n")
			writeFlags(&b, localFlags, c)
		}
		if hasVisibleFlags(inheritedFlags) {
			b.WriteString("\n")
			b.WriteString(helpHeading.Render("Global Flags:"))
			b.WriteString("\n")
			writeFlags(&b, inheritedFlags, c)
		}
	} else {
		// Root command: show all flags together
		allFlags := c.Flags()
		if hasVisibleFlags(allFlags) {
			b.WriteString("\n")
			b.WriteString(helpHeading.Render("Flags:"))
			b.WriteString("\n")
			writeFlags(&b, allFlags, c)
		}
	}

	// Footer
	if c.HasAvailableSubCommands() {
		b.WriteString("\n")
		b.WriteString(helpDesc.Render(fmt.Sprintf(
			"Use \"%s [command] --help\" for more information about a command.",
			c.CommandPath(),
		)))
		b.WriteString("\n")
	}

	fmt.Fprint(c.OutOrStdout(), b.String())
	return nil
}

func commandsInGroup(parent *cobra.Command, groupID string) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range parent.Commands() {
		if c.IsAvailableCommand() && c.GroupID == groupID {
			out = append(out, c)
		}
	}
	return out
}

func ungroupedCommands(parent *cobra.Command) []*cobra.Command {
	grouped := make(map[string]bool)
	for _, g := range parent.Groups() {
		grouped[g.ID] = true
	}
	var out []*cobra.Command
	for _, c := range parent.Commands() {
		if c.IsAvailableCommand() && !grouped[c.GroupID] && c.GroupID == "" {
			out = append(out, c)
		}
	}
	return out
}

func availableCommands(parent *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, c := range parent.Commands() {
		if c.IsAvailableCommand() {
			out = append(out, c)
		}
	}
	return out
}

func writeCommandList(b *strings.Builder, cmds []*cobra.Command) {
	// Find max name length for alignment
	maxLen := 0
	for _, c := range cmds {
		if len(c.Name()) > maxLen {
			maxLen = len(c.Name())
		}
	}
	for _, c := range cmds {
		b.WriteString("  ")
		b.WriteString(helpCommand.Render(fmt.Sprintf("%-*s", maxLen, c.Name())))
		b.WriteString("   ")
		b.WriteString(helpDesc.Render(c.Short))
		b.WriteString("\n")
	}
}

func writeFlags(b *strings.Builder, flags *pflag.FlagSet, c *cobra.Command) {
	// Collect visible flags and compute alignment
	type flagInfo struct {
		short    string
		name     string
		typeName string
		defVal   string
		usage    string
		required bool
	}

	var infos []flagInfo
	maxCol := 0

	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		fi := flagInfo{
			name:  f.Name,
			usage: f.Usage,
		}
		if f.Shorthand != "" {
			fi.short = "-" + f.Shorthand + ", "
		} else {
			fi.short = "    "
		}
		typeName := f.Value.Type()
		if typeName == "bool" {
			fi.typeName = ""
		} else {
			fi.typeName = " " + typeName
		}
		fi.defVal = f.DefValue
		if typeName != "bool" && fi.defVal != "" && fi.defVal != "0" && fi.defVal != "\"\"" && fi.defVal != "[]" {
			fi.usage += fmt.Sprintf(" (default %s)", fi.defVal)
		}

		// Check if required
		if ann, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			for _, v := range ann {
				if v == "true" {
					fi.required = true
				}
			}
		}

		col := len(fi.short) + 2 + len(fi.name) + len(fi.typeName) // "--" + name + type
		if col > maxCol {
			maxCol = col
		}
		infos = append(infos, fi)
	})

	for _, fi := range infos {
		b.WriteString("  ")

		flagStr := fmt.Sprintf("%s--%s%s", fi.short, fi.name, fi.typeName)
		b.WriteString(helpFlag.Render(flagStr))

		// Pad to align descriptions
		pad := maxCol - (len(fi.short) + 2 + len(fi.name) + len(fi.typeName)) + 3
		if pad < 1 {
			pad = 1
		}
		b.WriteString(strings.Repeat(" ", pad))

		if fi.required {
			b.WriteString(helpRequired.Render(fi.usage + " (required)"))
		} else {
			b.WriteString(helpDesc.Render(fi.usage))
		}
		b.WriteString("\n")
	}
}

func hasVisibleFlags(flags *pflag.FlagSet) bool {
	hasVisible := false
	flags.VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			hasVisible = true
		}
	})
	return hasVisible
}
