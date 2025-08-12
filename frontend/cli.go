package frontend

import (
	"errors"
	"fmt"
	"math"
	"spieven/common/packet"
	"spieven/common/types"
	"strconv"

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
			tags                     []string
		)
		cmd := &cobra.Command{
			Use:   "list [OPTIONS...]",
			Short: "Display a list of running tasks",
			RunE: func(cmd *cobra.Command, args []string) error {
				filter := packet.ListRequestBodyFilter{
					IdFilter:      idFilter,
					NameFilter:    nameFilter,
					AllTagsFilter: tags,
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
		cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Filter tasks by tags. Multiple tags can be specified (comma separated) to require multiple tags to be present")
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
			tags                   []string
		)

		longDescription := "Schedule a new task. By default all option arguments (starting with hyphens) will be interpreted " +
			"as Spieven arguments. To pass option arguments to the actual command to be scheduled, use a -- separator before " +
			"the command. Examples:" +
			"\n  spieven schedule notify-send 'Hello'                                       # OK: no ptions" +
			"\n  spieven schedule --delay-after-success 1000 notify-send 'Hello'            # OK: only Spieven options" +
			"\n  spieven schedule --delay-after-success 1000 notify-send 'Hello' -t 500     # Probably NOT OK: -t will be treated as a Spieven option" +
			"\n  spieven schedule --delay-after-success 1000 -- notify-send 'Hello' -t 500  # OK: -t will be treated as notify-send option" +
			"\n" +
			"\nThe scheduled task can be rejected by Spieven backend for a number of reasons, e.g. duplicate task. In that case the " +
			"schedule command will fail and print appropriate error string. The schedule command will succeed as long as Spieven " +
			"backend decides it can run the task. This doesn't neccessarily mean the task itself can succeed - it can fail immediately " +
			"or at any point in the future. In order to query the state of running task, use list or peek command."

		cmd := &cobra.Command{
			Use:   "schedule [OPTIONS...] [--] COMMAND [COMMAND_ARGS...]",
			Short: "Schedule a new task",
			Long:  longDescription,
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
						displaySelection, rerunDelayAfterSuccess, rerunDelayAfterFailure, maxSubsequentFailures, tags)
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
				return err
			},
		}
		cmd.Flags().StringVarP(&friendlyName, "friendly-name", "n", "", "A friendly name for the task. It will appear in various logs for easier identification. By default an executable name will be used.")
		cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")
		cmd.Flags().BoolVarP(&captureStdout, "capture-stdout", "c", false, "Capture stdout to a separate file. This is required to be able to query stdout contents later.")
		cmd.Flags().StringVarP(&display, "display", "p", "", "Force a specific display. "+types.DisplaySelectionHelpString)
		cmd.Flags().IntVarP(&rerunDelayAfterSuccess, "delay-after-success", "s", 0, "Delay in milliseconds before rerunning scheduled command after a successful execution")
		cmd.Flags().IntVarP(&rerunDelayAfterFailure, "delay-after-failure", "f", 0, "Delay in milliseconds before rerunning scheduled command after a failed execution")
		cmd.Flags().IntVarP(&maxSubsequentFailures, "max-subsequent-failures", "m", 3, "Specify a number of command failures in a row after which the task will become deactivated. Specify -1 for no limit.")
		cmd.Flags().StringSliceVarP(&tags, "tags", "t", []string{}, "Specify comma-separated list of tags for the task. Task do not have any effect, but they can be used to filter tasks.")

		commands = append(commands, cmd)
	}

	{
		cmd := &cobra.Command{
			Use:   "peek TASK_ID [OPTIONS...]",
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
			Use:   "refresh [TASK_ID]",
			Short: "Cancels a wait between task's command execution. If TASK_ID is not specified, all tasks are refreshed.",
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

	{
		var watch bool
		cmd := &cobra.Command{
			Use:   "reschedule TASK_ID [OPTIONS...]",
			Short: "Reschedule a deactivated task by its ID.",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				taskId, err := strconv.Atoi(args[0])
				if err != nil {
					return err
				}

				connection, err := ConnectToBackend()
				if err == nil {
					defer connection.Close()
					response, err := CmdReschedule(connection, taskId)
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
		cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch log file after successful scheduling. Functionally equivalent to running Spieven watch <taskId>")
		commands = append(commands, cmd)
	}

	return
}
