package frontend

import (
	"fmt"
	"math"
	"strconv"
	"supervisor/common/packet"
	"supervisor/common/types"

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
			display, err := cmd.Flags().GetString("display")
			if err != nil {
				return err
			}
			includeDeactivated, err := cmd.Flags().GetBool("include-deactivated")
			if err != nil {
				return err
			}
			jsonOutput, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			filter := packet.ListRequestBodyFilter{
				IdFilter:   idFilter,
				NameFilter: nameFilter,
			}
			if err := filter.DisplayFilter.ParseDisplaySelection(display); err != nil {
				return err
			}
			filter.Derive()

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				err = CmdList(connection, filter, includeDeactivated, jsonOutput)
			}
			return err
		},
	}
	listCmd.Flags().IntP("id", "i", math.MaxInt, "Filter tasks by id")
	listCmd.Flags().StringP("name", "n", "", "Filter tasks by friendly name")
	listCmd.Flags().StringP("display", "p", "", "Filter tasks by display. "+types.DisplaySelectionHelpString)
	listCmd.Flags().BoolP("include-deactivated", "d", false, "Include deactivated tasks as well as actively running ones")
	listCmd.Flags().BoolP("json", "j", false, "Display output as json.")
	// TODO Add -D option to always load deactivated and -d option to load deactivated only if not found

	scheduleCmd := &cobra.Command{
		// TODO add -- separator to allow passing dash args as a cmdline to run
		Use:   "schedule command [args...]",
		Short: "Schedule a new task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userIndex, err := cmd.Flags().GetInt("userIndex")
			if err != nil {
				return err
			}
			friendlyName, err := cmd.Flags().GetString("friendly-name")
			if err != nil {
				return err
			}
			watch, err := cmd.Flags().GetBool("watch")
			if err != nil {
				return err
			}
			captureStdout, err := cmd.Flags().GetBool("capture-stdout")
			if err != nil {
				return err
			}
			display, err := cmd.Flags().GetString("display")
			if err != nil {
				return err
			}
			var displaySelection types.DisplaySelection
			if err := displaySelection.ParseDisplaySelection(display); err != nil {
				return err
			}

			connection, err := ConnectToBackend()
			if err == nil {
				defer connection.Close()
				response, err := CmdSchedule(connection, args, userIndex, friendlyName, captureStdout, displaySelection)
				if err != nil {
					return err
				}

				if watch {
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
	scheduleCmd.Flags().StringP("friendly-name", "n", "", "A friendly name for the task. It will appear in various logs for easier identification. By default an executable name will be used.")
	scheduleCmd.Flags().BoolP("watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")
	scheduleCmd.Flags().BoolP("capture-stdout", "c", false, "Capture stdout to a separate file. This is required to be able to query stdout contents later.")
	scheduleCmd.Flags().StringP("display", "p", "", "Force a specific display. "+types.DisplaySelectionHelpString)

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

	// TODO add reschedule [taskId] command. We will have to rewrite the .ndjson file

	// TODO add check command to see whether the server is running and responds correctly

	// TODO add refresh command. Refresh all when no arg, allow filters like list.

	return []*cobra.Command{
		logCmd,
		listCmd,
		scheduleCmd,
		peekCmd,
	}
}
