package main

import (
	"fmt"
	"os"
	"supervisor/backend"
	"supervisor/frontend"
)

func main() {
	backendCmd := backend.CreateCliCommand()

	frontendCommands := frontend.CreateCliCommands()
	for _, cmd := range frontendCommands {
		backendCmd.AddCommand(cmd)
	}

	err := backendCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
