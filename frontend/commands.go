package frontend

import (
	"fmt"
	"net"
	"os"
	"supervisor/common"
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
	fmt.Printf("Running processes: %v\n", summary.ConnectionCount)
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
		fmt.Println("No processes are running")
		return nil
	}

	for i, process := range response {
		activeStr := "Yes"
		if process.IsDeactivated {
			activeStr = fmt.Sprintf("No (%v)", process.DeactivationReason)
		}

		fmt.Printf("Active:                %v\n", activeStr)
		fmt.Printf("Id:                    %v\n", process.Id)
		fmt.Printf("Cmdline:               %v\n", process.Cmdline)
		fmt.Printf("Cwd:                   %v\n", process.Cwd)
		fmt.Printf("OutFilePath:           %v\n", process.OutFilePath)
		fmt.Printf("MaxSubsequentFailures: %v\n", process.MaxSubsequentFailures)
		fmt.Printf("UserIndex:             %v\n", process.UserIndex)

		if i < len(response)-1 {
			fmt.Println()
		}
	}

	return nil
}

func CmdRegister(backendConnection net.Conn, args []string, userIndex int) error {
	cwd, err := os.Getwd()
	if err != nil {
		var found bool
		cwd, found = os.LookupEnv("HOME")
		if !found {
			return fmt.Errorf("could not determine working directory for the task")
		}
	}

	body := common.RegisterBody{
		Cmdline:   args,
		Cwd:       cwd,
		Env:       os.Environ(),
		UserIndex: userIndex,
	}

	requestPacket, err := common.EncodeRegisterPacket(body)
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

	registered, err := common.DecodeRegisterResponsePacket(responsePacket)
	if err != nil {
		return err
	}

	if registered {
		fmt.Println("Process registered")
	} else {
		fmt.Println("Process already running")
	}

	return nil
}
