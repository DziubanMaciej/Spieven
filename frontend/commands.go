package frontend

import (
	"errors"
	"fmt"
	"net"
	"os"
	"supervisor/common"
	"sync"
	"sync/atomic"
)

func CmdSummary(backendConnection net.Conn) error {
	requestPacket, err := common.EncodeSummaryPacket()
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

	summary, err := common.DecodeSummaryResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	fmt.Printf("Version: %v\n", summary.Version)
	fmt.Printf("Running tasks: %v\n", summary.ConnectionCount)
	return nil
}

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

func CmdList(backendConnection net.Conn) error {
	requestPacket, err := common.EncodeListPacket()
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
	requestPacket, err := common.EncodeNotifyTaskEndPacket(taskId)
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, requestPacket)
	if err != nil {
		return err
	}

	var sync sync.WaitGroup
	sync.Add(2)

	// Goroutine 1: Read the file continuously
	var watchFileStopFlag atomic.Int32
	go func() {
		defer sync.Done()

		err = WatchFile(logFilePath, &watchFileStopFlag)
		if err != nil {
			// TODO implement this. Try using SetReadDeadline on the socket. Research if it needs to be reset to some default afterwards.
			fmt.Println("Error watching file. Currently stopping another goroutine is not implemented, so this process will hang until the task ends.")
		}
	}()

	// Goroutine 2: Wait for the response packet
	var backendReceiveErr error
	go func() {
		defer sync.Done()
		defer func() { watchFileStopFlag.Store(1) }()

		responsePacket, err := common.ReceivePacket(backendConnection)
		if err != nil {
			backendReceiveErr = err
			return
		}

		response, err := common.DecodeNotifyTaskEndResponsePacket(responsePacket)
		if err != nil {
			backendReceiveErr = err
			return
		}

		switch response {
		case common.NotifyTaskEndResponseEnded:
			return
		case common.NotifyTaskEndResponseInvalidTask:
			backendReceiveErr = errors.New("invalid task ID sent to backend")
			return
		default:
			backendReceiveErr = errors.New("unknown backend error")
			return
		}
	}()

	sync.Wait()
	if backendReceiveErr != nil {
		fmt.Println(backendReceiveErr)
	}

	return nil
}
