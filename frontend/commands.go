package frontend

import (
	"fmt"
	"net"
	"supervisor/common"
)

func CmdSummary(backendConnection net.Conn) error {
	request_packet, err := common.EncodeSummaryPacket()
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, request_packet)
	if err != nil {
		return err
	}

	response_packet, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	summary, err := common.DecodeSummaryResponsePacket(response_packet)
	if err != nil {
		return err
	}

	fmt.Printf("Version: %v\n", summary.Version)
	fmt.Printf("Running processes: %v\n", summary.ConnectionCount)
	return nil
}

func CmdList(backendConnection net.Conn) error {
	request_packet, err := common.EncodeListPacket()
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, request_packet)
	if err != nil {
		return err
	}

	response_packet, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	response, err := common.DecodeListResponsePacket(response_packet)
	if err != nil {
		return err
	}

	if len(response) == 0 {
		fmt.Println("No processes are running")
		return nil
	}

	for i, process := range response {
		fmt.Printf("Id:                    %v\n", process.Id)
		fmt.Printf("Cmdline:               %v\n", process.Cmdline)
		fmt.Printf("Cwd:                   %v\n", process.Cwd)
		fmt.Printf("OutFilePath:           %v\n", process.OutFilePath)
		fmt.Printf("MaxSubsequentFailures: %v\n", process.MaxSubsequentFailures)

		if i < len(response)-1 {
			fmt.Println()
		}
	}

	return nil
}

func CmdRegister(backendConnection net.Conn, args []string) error {
	body := common.RegisterBody{
		Cmdline: args,
		Cwd:     "",
	}

	packet, err := common.EncodeRegisterPacket(body)
	if err != nil {
		return err
	}

	err = common.SendPacket(backendConnection, packet)
	if err != nil {
		return err
	}

	response_packet, err := common.ReceivePacket(backendConnection)
	if err != nil {
		return err
	}

	registered, err := common.DecodeRegisterResponsePacket(response_packet)
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
