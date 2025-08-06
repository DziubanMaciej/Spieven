package frontend

import (
	"fmt"
	"math"
	"strconv"
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
			idFilter, err := cmd.Flags().GetInt("id")
			if err != nil {
				return err
			}
			nameFilter, err := cmd.Flags().GetString("name")
			if err != nil {
				return err
			}
			includeDeactivated, err := cmd.Flags().GetBool("includeDeactivated")
			if err != nil {
				return err
			}
			displayResult, err := cmd.Flags().GetBool("result")
			if err != nil {
				return err
			}
			filter := common.ListFilter{
				IdFilter:   idFilter,
				NameFilter: nameFilter,
			}
			if err := filter.Derive(); err != nil {
				return err
			}

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = CmdList(connection, filter, includeDeactivated, displayResult)
			}
			return err
		},
	}
	listCmd.Flags().IntP("id", "i", math.MaxInt, "Filter tasks by id")
	listCmd.Flags().StringP("name", "n", "", "Filter tasks by friendly name")
	listCmd.Flags().BoolP("includeDeactivated", "d", false, "Include deactivated tasks as well as actively running ones")
	listCmd.Flags().BoolP("result", "r", false, "Display last exit value and stdout of the task. Must be used with either id or name filter.")

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
			captureStdout, err := cmd.Flags().GetBool("captureStdout")
			if err != nil {
				return err
			}

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				response, err := CmdSchedule(connection, args, userIndex, friendlyName, captureStdout)
				if err != nil {
					return err
				}

				if watch {
					// TODO do not do  this if failed to schedule
					err := CmdWatchTaskLog(connection, response.Id, &response.LogFile)
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
	scheduleCmd.Flags().BoolP("captureStdout", "c", false, "Capture stdout to a separate file. This is required to be able to query stdout contents later.")

	peekCmd := &cobra.Command{
		Use:   "peek [taskID]",
		Short: "Displays logs of a given task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid integer: %v", err)
			}

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = CmdWatchTaskLog(connection, taskId, nil)
			}
			return err
		},
	}

	return []*cobra.Command{
		logCmd,
		listCmd,
		scheduleCmd,
		peekCmd,
	}
}
