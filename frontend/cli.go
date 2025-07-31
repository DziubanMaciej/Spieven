package frontend

import (
	"fmt"
	"math"
	"supervisor/common"

	"github.com/spf13/cobra"
)

func CreateCliCommands() []*cobra.Command {
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Display a backend log",
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = CmdLog(connection)
			}
			return err
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Display a list of running tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmd.Flags().GetUint32("id")
			if err != nil {
				return err
			}

			includeDeactivated, err := cmd.Flags().GetBool("includeDeactivated")
			if err != nil {
				return err
			}

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = CmdList(connection, id, includeDeactivated)
			}
			return err
		},
	}
	listCmd.Flags().Uint32P("id", "i", math.MaxUint32, "Display a task with a specific ID")
	listCmd.Flags().BoolP("includeDeactivated", "d", false, "Include deactivated tasks as well as actively running ones")

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

	scheduleCmd := &cobra.Command{
		Use:   "schedule command [args...]",
		Short: "Schedule a new task",
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

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				response, err := CmdSchedule(connection, args, userIndex, friendlyName)
				if err != nil {
					return err
				}

				if watch {
					err := CmdWatchTaskLog(connection, response.Id, response.LogFile)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	scheduleCmd.Flags().IntP("userIndex", "i", 0, "An index used to differentiate between different tasks with the same settings. Does not serve any purpose other than to allow for duplicate tasks running.")
	scheduleCmd.Flags().StringP("friendlyName", "n", "", "A friendly name for the task. It will appear in various logs for easier identification. By default an executable name will be used.")
	scheduleCmd.Flags().BoolP("watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")

	// TODO add "watchLog" command. For that we'll need a way to query logFile by taskId. This can get tricky if the task is no longer running.

	return []*cobra.Command{
		logCmd,
		listCmd,
		probeX11Cmd, // TODO move this to some "internal" package
		scheduleCmd,
	}
}
