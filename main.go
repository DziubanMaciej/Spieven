package main

import (
	"fmt"
	"os"
	"spieven/backend"
	"spieven/common"
	"spieven/common/buildopts"
	"spieven/frontend"
	"spieven/internal"

	"github.com/spf13/cobra"
)

func main() {
	buildopts.PrintBuildFlavourNotice()

	rootCmd := &cobra.Command{
		Use:          "spieven",
		Short:        "Spieven is a process spieven for Linux",
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	backendCmd := backend.CreateCliCommand()
	rootCmd.AddCommand(backendCmd)

	frontendCommands := frontend.CreateCliCommands()
	for _, cmd := range frontendCommands {
		rootCmd.AddCommand(cmd)
	}

	internalCommand := internal.CreateCliCommands()
	rootCmd.AddCommand(internalCommand)

	{
		cmd := &cobra.Command{
			Use:   "version",
			Short: "Print the version number",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(common.Version)
			},
		}
		rootCmd.AddCommand(cmd)

	}

	common.CliApplyRecursively(rootCmd, common.CliSetPassthroughUsage)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
