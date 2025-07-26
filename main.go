package main

import (
	"fmt"
	"os"
	"supervisor/backend"
	"supervisor/frontend"

	"github.com/spf13/cobra"
)

func main() {
	noParamsCmd := &cobra.Command{
		Use:   "app",
		Short: "A CLI tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return backend.RunServer()
		},
	}

	summaryCmd := &cobra.Command{
		Use:   "summary",
		Short: "Show a summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = frontend.CmdSummary(connection)
			}
			return err
		},
	}
	noParamsCmd.AddCommand(summaryCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Display a list of running processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = frontend.CmdList(connection)
			}
			return err
		},
	}
	noParamsCmd.AddCommand(listCmd)

	registerCmd := &cobra.Command{
		Use:   "register command [args...]",
		Short: "Register process to execute",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = frontend.CmdRegister(connection, args)
			}
			return err
		},
	}
	noParamsCmd.AddCommand(registerCmd)

	err := noParamsCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
