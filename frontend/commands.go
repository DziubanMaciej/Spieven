package frontend

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"spieven/common/packet"
	"spieven/common/types"
	"sync"
	"sync/atomic"
	"time"
)

func CmdLog(backendConnection net.Conn) error {
	requestPacket, err := packet.EncodeLogPacket()
	if err != nil {
		return err
	}

	err = packet.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	responsePacket, err := packet.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := packet.DecodeLogResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	for _, line := range response {
		fmt.Println(line)
	}

	return nil
}

func CmdList(
	backendConnection net.Conn,
	filter packet.ListRequestBodyFilter,
	includeDeactivated bool,
	includeDeactivatedAlways bool,
	jsonOutput bool) error {
	request := packet.ListRequestBody{
		Filter:                   filter,
		IncludeDeactivated:       includeDeactivated,
		IncludeDeactivatedAlways: includeDeactivatedAlways,
	}

	requestPacket, err := packet.EncodeListPacket(request)
	if err != nil {
		return err
	}

	err = packet.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	responsePacket, err := packet.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := packet.DecodeListResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	if len(response) == 0 {
		if filter.HasAnyFilter {
			return errors.New("no tasks match the requested criteria")
		} else {
			return errors.New("no tasks found")
		}
	}

	if jsonOutput {
		output, err := json.MarshalIndent(response, "", "    ")
		if err != nil {
			return errors.New("task has not finished execution yet")
		}
		fmt.Println(string(output))
	} else {
		for i, task := range response {
			activeStr := "Yes"
			if task.IsDeactivated {
				activeStr = fmt.Sprintf("No (%v)", task.DeactivationReason)
			}

			fmt.Printf("Task %v\n", task.FriendlyName)
			fmt.Printf("  Active:                 %v\n", activeStr)
			fmt.Printf("  Id:                     %v\n", task.Id)
			fmt.Printf("  Cmdline:                %v\n", task.Cmdline)
			fmt.Printf("  Cwd:                    %v\n", task.Cwd)
			fmt.Printf("  Display:                %v\n", task.Display.ComputeDisplayLabelLong())
			fmt.Printf("  OutFilePath:            %v\n", task.OutFilePath)
			fmt.Printf("  MaxSubsequentFailures:  %v\n", task.MaxSubsequentFailures)
			fmt.Printf("  RunCount:               %v\n", task.RunCount)
			fmt.Printf("  FailureCount:           %v\n", task.FailureCount)
			fmt.Printf("  SubsequentFailureCount: %v\n", task.SubsequentFailureCount)
			fmt.Printf("  LastExitValue:          %v\n", task.LastExitValue)
			if i < len(response)-1 {
				fmt.Println()
			}
		}
	}

	return nil
}

func CmdSchedule(
	backendConnection net.Conn,
	args []string,
	friendlyName string,
	captureStdout bool,
	display types.DisplaySelection,
	rerunDelayAfterSuccess int,
	rerunDelayAfterFailure int,
	maxSubsequentFailures int) (*packet.ScheduleResponseBody, error) {
	cwd, err := os.Getwd()
	if err != nil {
		var found bool
		cwd, found = os.LookupEnv("HOME")
		if !found {
			return nil, fmt.Errorf("could not determine working directory for the task")
		}
	}

	if friendlyName == "" {
		friendlyName = args[0]
	}

	body := packet.ScheduleRequestBody{
		Cmdline:               args,
		Cwd:                   cwd,
		Env:                   os.Environ(),
		FriendlyName:          friendlyName,
		CaptureStdout:         captureStdout,
		Display:               display,
		DelayAfterSuccessMs:   rerunDelayAfterSuccess,
		DelayAfterFailureMs:   rerunDelayAfterFailure,
		MaxSubsequentFailures: maxSubsequentFailures,
	}

	err = ValidateScheduleRequestBody(&body)
	if err != nil {
		return nil, err
	}

	requestPacket, err := packet.EncodeSchedulePacket(body)
	if err != nil {
		return nil, err
	}

	err = packet.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return nil, err
	}

	responsePacket, err := packet.ReceivePacket(backendConnection)
	if err != nil {
		return nil, err
	}

	response, err := packet.DecodeScheduleResponsePacket(responsePacket)
	if err != nil {
		return nil, err
	}

	switch response.Status {
	case types.ScheduleResponseStatusSuccess:
		fmt.Println("Scheduled task")
		fmt.Println("Log file: ", response.LogFile)
		return &response, nil
	case types.ScheduleResponseStatusAlreadyRunning:
		err = errors.New("task is already scheduled. To run multiple instances of the same task use friendly name. See help message for details")
		return nil, err
	case types.ScheduleResponseStatusNameDisplayAlreadyRunning:
		err = fmt.Errorf("task named %v is already running on current display", friendlyName)
		return nil, err
	case types.ScheduleResponseStatusInvalidDisplay:
		err = errors.New("task is using invalid display")
		return nil, err
	default:
		err = errors.New("unknown scheduling error")
		return nil, err
	}
}

func CmdWatchTaskLog(backendConnection net.Conn, taskId int, logFilePath *string) error {
	retrieveLogFilePath := func() (string, error) {
		filter := packet.ListRequestBodyFilter{IdFilter: taskId}
		request := packet.ListRequestBody{
			Filter:             filter,
			IncludeDeactivated: true,
		}

		requestPacket, err := packet.EncodeListPacket(request)
		if err != nil {
			return "", err
		}

		err = packet.SendPacket(backendConnection, requestPacket)
		if err != nil {
			return "", err
		}

		responsePacket, err := packet.ReceivePacket(backendConnection)
		if err != nil {
			return "", err
		}

		response, err := packet.DecodeListResponsePacket(responsePacket)
		if err != nil {
			return "", err
		}

		switch len(response) {
		case 0:
			return "", fmt.Errorf("could not find log file")
		case 1:
			return response[0].OutFilePath, nil
		default:
			return "", fmt.Errorf("multiple log files found. This is highly unexpected")
		}
	}

	checkTaskActiveStatus := func() (bool, error) {
		requestPacket, err := packet.EncodeQueryTaskActivePacket(packet.QueryTaskActiveRequestBody(taskId))
		if err != nil {
			return false, err
		}

		err = packet.SendPacket(backendConnection, requestPacket)
		if err != nil {
			return false, err
		}

		responsePacket, err := packet.ReceivePacket(backendConnection)
		if err != nil {
			return false, err
		}

		response, err := packet.DecodeQueryTaskActiveResponsePacket(responsePacket)
		if err != nil {
			return false, err
		}

		switch response {
		case packet.QueryTaskActiveResponseBodyActive:
			return true, nil
		case packet.QueryTaskActiveResponseBodyInactive:
			return false, nil
		case packet.QueryTaskActiveResponseInvalidTask:
			return false, errors.New("invalid task ID sent to backend")
		default:
			return false, errors.New("unknown backend error")
		}
	}

	// If we don't know the path to the log file, we can ask backend for it using taskId.
	if logFilePath == nil {
		path, err := retrieveLogFilePath()
		if err != nil {
			return err
		}
		logFilePath = &path
	}

	// Setup vars for communicating with goroutines
	var goroutinesStopFlag atomic.Int32
	var sync sync.WaitGroup
	sync.Add(2)

	// Goroutine 1: Read the file continuously
	var fileWatchError error
	go func() {
		fileWatchError = WatchFile(taskId, *logFilePath, &goroutinesStopFlag)

		sync.Done()
		goroutinesStopFlag.Store(1)
	}()

	// Goroutine 2: Wait for the response packet
	var backendCommunicationError error
	go func() {
		taskActive := true
		for goroutinesStopFlag.Load() == 0 && taskActive {
			taskActive, backendCommunicationError = checkTaskActiveStatus()
			if taskActive {
				time.Sleep(time.Second)
			}
		}

		sync.Done()
		goroutinesStopFlag.Store(1)
	}()

	// Wait for both goroutines
	sync.Wait()

	// Print error if any
	if fileWatchError != nil {
		fmt.Printf("Log file watching error: %v\n", fileWatchError)
	}
	if backendCommunicationError != nil {
		fmt.Printf("Backend communication error: %v\n", backendCommunicationError)
	}

	return nil
}

func CmdRefresh(backendConnection net.Conn, taskId int) error {
	request := packet.RefreshBody{
		TaskId: taskId,
	}

	requestPacket, err := packet.EncodeRefreshPacket(request)
	if err != nil {
		return err
	}

	err = packet.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	responsePacket, err := packet.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := packet.DecodeRefreshResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	if taskId == -1 {
		// We asked to refresh all tasks
		if response.ActiveTasksCount != response.RefreshedTasksCount {
			return errors.New("not all tasks were refreshed")
		} else {
			fmt.Println("all tasks refreshed")
			return nil
		}
	} else {
		// We asked to refresh a specific task
		switch response.RefreshedTasksCount {
		case 0:
			return errors.New("invalid task id")
		case 1:
			fmt.Println("task refreshed")
			return nil
		default:
			return errors.New("multiple tasks refreshed, this is unexpected")
		}
	}
}

func CmdReschedule(backendConnection net.Conn, taskId int) (*packet.RescheduleResponseBody, error) {
	request := packet.RescheduleRequestBody{
		TaskId: taskId,
	}

	requestPacket, err := packet.EncodeReschedulePacket(request)
	if err != nil {
		return nil, err
	}

	err = packet.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return nil, err
	}

	responsePacket, err := packet.ReceivePacket(backendConnection)
	if err != nil {
		return nil, err
	}

	response, err := packet.DecodeRescheduleResponsePacket(responsePacket)
	if err != nil {
		return nil, err
	}

	switch response.Status {
	case types.ScheduleResponseStatusSuccess:
		fmt.Println("Rescheduled task")
		fmt.Println("Log file: ", response.LogFile)
		return &response, nil
	case types.ScheduleResponseStatusAlreadyRunning:
		err = fmt.Errorf("task is already scheduled. Looks like you scheduled an identical task after task %v was deactivated", taskId)
		return nil, err
	case types.ScheduleResponseStatusNameDisplayAlreadyRunning:
		err = fmt.Errorf("task with this name is already running on current display. Looks like you scheduled an identical task after task %v was deactivated", taskId)
		return nil, err
	case types.ScheduleResponseStatusInvalidDisplay:
		err = errors.New("task is using invalid display")
		return nil, err
	case types.ScheduleResponseStatusTaskNotFound:
		err = errors.New("task not found")
		return nil, err
	case types.ScheduleResponseStatusTaskNotDeactivated:
		err = errors.New("task is already active")
		return nil, err
	default:
		err = errors.New("unknown rescheduling error")
		return nil, err
	}
}
