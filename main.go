package main

import (
	"fmt"
	"os"
	"supervisor/backend"
	"supervisor/frontend"
	"supervisor/internal"
)

func main() {
	backendCmd := backend.CreateCliCommand()

	frontendCommands := frontend.CreateCliCommands()
	for _, cmd := range frontendCommands {
		backendCmd.AddCommand(cmd)
	}

	// TODO do not display help message all the time

	internalCommand := internal.CreateCliCommands()
	backendCmd.AddCommand(internalCommand)

	err := backendCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
