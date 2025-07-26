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
			userIndex, err := cmd.Flags().GetInt("userIndex")
			if err != nil {
				return err
			}

			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = frontend.CmdRegister(connection, args, userIndex)
			}
			return err
		},
	}
	registerCmd.Flags().IntP("userIndex", "i", 0, "An index used to differentiate between different processes with the same settings. Does not serve any purpose other than to allow for duplicate processes running.")
	noParamsCmd.AddCommand(registerCmd)

	err := noParamsCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
