package frontend

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"supervisor/common/packet"
	"supervisor/common/types"

	"github.com/spf13/cobra"
)

func CreateCliCommands() (commands []*cobra.Command) {
	{
		cmd := &cobra.Command{
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
		commands = append(commands, cmd)
	}

	{
		var (
			idFilter                 int
			nameFilter               string
			display                  string
			includeDeactivated       bool
			includeDeactivatedAlways bool
			jsonOutput               bool
		)
		cmd := &cobra.Command{
			Use:   "list",
			Short: "Display a list of running tasks",
			RunE: func(cmd *cobra.Command, args []string) error {
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
					err = CmdList(connection, filter, includeDeactivated, includeDeactivatedAlways, jsonOutput)
				}
				return err
			},
		}
		cmd.Flags().IntVarP(&idFilter, "id", "i", math.MaxInt, "Filter tasks by id")
		cmd.Flags().StringVarP(&nameFilter, "name", "n", "", "Filter tasks by friendly name")
		cmd.Flags().StringVarP(&display, "display", "p", "", "Filter tasks by display. "+types.DisplaySelectionHelpString)
		cmd.Flags().BoolVarP(&includeDeactivated, "include-deactivated", "d", false, "Include deactivated tasks if no tasks were found among active ones")
		cmd.Flags().BoolVarP(&includeDeactivatedAlways, "include-deactivated-always", "D", false, "Include deactivated tasks as well as active ones")
		cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Display output as json.")
		commands = append(commands, cmd)
	}

	{
		var (
			friendlyName           string
			watch                  bool
			captureStdout          bool
			display                string
			rerunDelayAfterSuccess int
			rerunDelayAfterFailure int
			maxSubsequentFailures  int
		)
		cmd := &cobra.Command{
			// TODO add -- separator to allow passing dash args as a cmdline to run
			Use:   "schedule command [args...]",
			Short: "Schedule a new task",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var displaySelection types.DisplaySelection
				if err := displaySelection.ParseDisplaySelection(display); err != nil {
					return err
				}

				connection, err := ConnectToBackend()
				if err == nil {
					defer connection.Close()
					response, err := CmdSchedule(connection, args, friendlyName, captureStdout,
						displaySelection, rerunDelayAfterSuccess, rerunDelayAfterFailure, maxSubsequentFailures)
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
		cmd.Flags().StringVarP(&friendlyName, "friendly-name", "n", "", "A friendly name for the task. It will appear in various logs for easier identification. By default an executable name will be used.")
		cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")
		cmd.Flags().BoolVarP(&captureStdout, "capture-stdout", "c", false, "Capture stdout to a separate file. This is required to be able to query stdout contents later.")
		cmd.Flags().StringVarP(&display, "display", "p", "", "Force a specific display. "+types.DisplaySelectionHelpString)
		cmd.Flags().IntVarP(&rerunDelayAfterSuccess, "delay-after-success", "s", 0, "Delay in milliseconds before rerunning scheduled command after a successful execution")
		cmd.Flags().IntVarP(&rerunDelayAfterFailure, "delay-after-failure", "f", 0, "Delay in milliseconds before rerunning scheduled command after a failed execution")
		cmd.Flags().IntVarP(&maxSubsequentFailures, "max-subsequent-failures", "m", 3, "Specify a number of command failures in a row after which the task will become deactivated. Specify -1 for no limit.")
		commands = append(commands, cmd)
	}

	{
		cmd := &cobra.Command{
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
		commands = append(commands, cmd)
	}

	{
		cmd := &cobra.Command{
			Use:   "check",
			Short: "Checks whether the backend is running and can be connected to",
			RunE: func(cmd *cobra.Command, args []string) error {
				connection, err := ConnectToBackend()
				if err == nil {
					defer connection.Close()
					fmt.Println("backend works correctly")
					return nil
				}
				return errors.New("cannot connect to backend")
			},
		}
		commands = append(commands, cmd)
	}

	{
		cmd := &cobra.Command{
			Use:   "refresh",
			Short: "Cancels a wait",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				taskId := -1
				if len(args) > 0 {
					var err error
					taskId, err = strconv.Atoi(args[0])
					if err != nil {
						return err
					}
				}

				connection, err := ConnectToBackend()
				if err != nil {
					return errors.New("cannot connect to backend")
				}
				defer connection.Close()
				return CmdRefresh(connection, taskId)
			},
		}
		commands = append(commands, cmd)
	}

	// TODO add reschedule [taskId] command. We will have to rewrite the .ndjson file

	return
}
