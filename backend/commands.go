package backend

import (
	"fmt"
	"net"
	"supervisor/common"
)

func CmdSummary(backendState *BackendState, frontendConnection net.Conn) error {
	fmt.Printf("Summary\n")

	response := common.SummaryResponseBody{
		Version:         "1.0",
		ConnectionCount: 4,
	}

	packet, err := common.EncodeSummaryResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}

func CmdRegister(backendState *BackendState, frontendConnection net.Conn, request common.RegisterBody) error {
	process_description := ProcessDescription{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		OutFilePath:           "/home/maciej/work/Spieven/test_scripts/log.txt",
		MaxSubsequentFailures: 3,
	}

	registered := backendState.processes.TryRegisterProcess(&process_description, &backendState.messages)
	if registered {
		backendState.messages.AddF(BackendMessageInfo, "Registered process %v", request.Cmdline)
	} else {
		backendState.messages.Add(BackendMessageInfo, "Did not register process, because it's already running")
	}

	packet, err := common.EncodeRegisterResponsePacket(registered)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}
