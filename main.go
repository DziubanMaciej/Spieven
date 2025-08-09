package main

import (
	"os"
	"supervisor/backend"
	"supervisor/frontend"
	"supervisor/internal"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "Spieven",
		Short:        "Spieven is a process supervisor for Linux",
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

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
