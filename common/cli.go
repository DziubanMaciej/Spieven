package common

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

func CliApplyRecursively(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, subCmd := range cmd.Commands() {
		CliApplyRecursively(subCmd, fn)
	}
}

func CliSetPassthroughUsage(cmd *cobra.Command) {
	cmd.DisableFlagsInUseLine = true
	cmd.SetUsageFunc(CliUsageFunc)
}

func CliUsageFunc(c *cobra.Command) error {
	w := c.OutOrStderr()

	const helpCommandName = "help"
	rpad := func(s string, padding int) string {
		formattedString := fmt.Sprintf("%%-%ds", padding)
		return fmt.Sprintf(formattedString, s)
	}

	fmt.Fprint(w, "Usage:")
	if c.Runnable() {
		fmt.Fprintf(w, "\n  %s", c.UseLine())
	}
	if c.HasAvailableSubCommands() {
		cmds := c.Commands()
		for _, subcmd := range cmds {
			if subcmd.IsAvailableCommand() || subcmd.Name() == helpCommandName {
				fmt.Fprintf(w, "\n  %s %s", rpad(subcmd.Name(), subcmd.NamePadding()), subcmd.Short)
			}
		}

	}
	if c.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "\n\nFlags:\n")
		fmt.Fprint(w, strings.TrimRightFunc(c.LocalFlags().FlagUsages(), unicode.IsSpace))
	}
	if c.HasAvailableSubCommands() {
		fmt.Fprintf(w, "\n\nUse \"%s COMMAND --help\" for more information about available commands.", c.CommandPath())
	}
	fmt.Fprintln(w)
	return nil
}
