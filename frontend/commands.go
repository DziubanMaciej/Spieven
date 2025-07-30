package frontend

import (
	"errors"
	"fmt"
	"net"
	"os"
	"supervisor/common"
	"sync"
	"sync/atomic"
	"time"
)

func CmdLog(backendConnection net.Conn) error {
	requestPacket, err := common.EncodeLogPacket()
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	responsePacket, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := common.DecodeLogResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	for _, line := range response {
		fmt.Println(line)
	}

	return nil
}

func CmdList(backendConnection net.Conn, includeDeactivated bool) error {
	request := common.ListBody{
		IncludeDeactivated: includeDeactivated,
	}

	requestPacket, err := common.EncodeListPacket(request)
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	responsePacket, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := common.DecodeListResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	if len(response) == 0 {
		fmt.Println("No tasks are running")
		return nil
	}

	for i, task := range response {
		activeStr := "Yes"
		if task.IsDeactivated {
			activeStr = fmt.Sprintf("No (%v)", task.DeactivationReason)
		}

		fmt.Printf("Task %v\n", task.FriendlyName)
		fmt.Printf("  Active:                %v\n", activeStr)
		fmt.Printf("  Id:                    %v\n", task.Id)
		fmt.Printf("  Cmdline:               %v\n", task.Cmdline)
		fmt.Printf("  Cwd:                   %v\n", task.Cwd)
		fmt.Printf("  OutFilePath:           %v\n", task.OutFilePath)
		fmt.Printf("  MaxSubsequentFailures: %v\n", task.MaxSubsequentFailures)
		fmt.Printf("  UserIndex:             %v\n", task.UserIndex)

		if i < len(response)-1 {
			fmt.Println()
		}
	}

	return nil
}

func CmdSchedule(backendConnection net.Conn, args []string, userIndex int, friendlyName string) (*common.ScheduleResponseBody, error) {
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

	body := common.ScheduleBody{
		Cmdline:      args,
		Cwd:          cwd,
		Env:          os.Environ(),
		UserIndex:    userIndex,
		FriendlyName: friendlyName,
	}

	requestPacket, err := common.EncodeSchedulePacket(body)
	if err != nil {
		return nil, err
	}

	err = common.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return nil, err
	}

	responsePacket, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return nil, err
	}

	response, err := common.DecodeScheduleResponsePacket(responsePacket)
	if err != nil {
		return nil, err
	}

	switch response.Status {
	case common.ScheduleResponseSuccess:
		fmt.Println("Scheduled task")
		fmt.Println("Log file: ", response.LogFile)
	case common.ScheduleResponseAlreadyRunning:
		fmt.Println("Task is already scheduled. To run multiple instances of the same task use userIndex. See help message for details.")
	case common.ScheduleResponseInvalidDisplay:
		fmt.Println("Task is using invalid display")
	default:
		fmt.Println("Unknown scheduling error")
	}

	return &response, nil
}

func CmdWatchTaskLog(backendConnection net.Conn, taskId int, logFilePath string) error {
	checkTaskActiveStatus := func() (bool, error) {
		requestPacket, err := common.EncodeQueryTaskActivePacket(taskId)
		if err != nil {
			return false, err
		}

		err = common.SendPacket(backendConnection, requestPacket)
		if err != nil {
			return false, err
		}

		responsePacket, err := common.ReceivePacket(backendConnection)
		if err != nil {
			return false, err
		}

		response, err := common.DecodeQueryTaskActiveResponsePacket(responsePacket)
		if err != nil {
			return false, err
		}

		switch response {
		case common.QueryTaskActiveResponseBodyActive:
			return true, nil
		case common.QueryTaskActiveResponseBodyInactive:
			return false, nil
		case common.QueryTaskActiveResponseInvalidTask:
			return false, errors.New("invalid task ID sent to backend")
		default:
			return false, errors.New("unknown backend error")
		}
	}

	// Setup vars for communicating with goroutines
	var goroutinesStopFlag atomic.Int32
	var sync sync.WaitGroup
	sync.Add(2)

	// Goroutine 1: Read the file continuously
	var fileWatchError error
	go func() {
		fileWatchError = WatchFile(logFilePath, &goroutinesStopFlag)

		sync.Done()
		goroutinesStopFlag.Store(1)
	}()

	// Goroutine 2: Wait for the response packet
	var backendCommunicationError error
	go func() {
		taskActive := true
		for goroutinesStopFlag.Load() == 0 && taskActive {
			taskActive, backendCommunicationError = checkTaskActiveStatus() // TODO it's possible the file does not exist yet
			time.Sleep(time.Second)
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
