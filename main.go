package main

import (
	"fmt"
	"os"
	"supervisor/backend"
	"supervisor/common"
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

	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Display a backend log",
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = frontend.CmdLog(connection)
			}
			return err
		},
	}
	noParamsCmd.AddCommand(logCmd)

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

	probeX11Cmd := &cobra.Command{
		Use:           "watchxorg [display]",
		Hidden:        true,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dpyName := args[0]
			dpy := common.TryConnectXorg(dpyName)
			if dpy == nil {
				return fmt.Errorf("could not connect to xorg %v", dpyName)
			}

			fmt.Printf("Connected to xorg %v\n", dpyName)
			common.WatchXorgActive(dpy)
			common.DisconnectXorg(dpy)
			fmt.Printf("Disconnected from xorg %v\n", dpyName)
			return nil
		},
	}
	noParamsCmd.AddCommand(probeX11Cmd)

	registerCmd := &cobra.Command{
		Use:   "register command [args...]",
		Short: "Register process to execute",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userIndex, err := cmd.Flags().GetInt("userIndex")
			if err != nil {
				return err
			}
			friendlyName, err := cmd.Flags().GetString("friendlyName")
			if err != nil {
				return err
			}
			watch, err := cmd.Flags().GetBool("watch")
			if err != nil {
				return err
			}

			connection, err := frontend.ConnectToBackend()
			if err == nil {
				defer connection.Close()
				response, err := frontend.CmdRegister(connection, args, userIndex, friendlyName)
				if err != nil {
					return err
				}

				if watch {
					err := frontend.CmdWatchTaskLog(connection, response.Id, response.LogFile)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	registerCmd.Flags().IntP("userIndex", "i", 0, "An index used to differentiate between different processes with the same settings. Does not serve any purpose other than to allow for duplicate processes running.")
	registerCmd.Flags().StringP("friendlyName", "n", "", "A friendly name for the task. It will appear in various logs for easier identification. By default an executable name will be used.")
	registerCmd.Flags().BoolP("watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")
	noParamsCmd.AddCommand(registerCmd)

	// TODO add "watchLog" command. For that we'll need a way to query logFile by taskId. This can get tricky if the task is no longer running.

	err := noParamsCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
