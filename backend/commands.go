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

	err = common.SendPacket(frontendConnection, packet)
	if err != nil {
		return err
	}

	return nil
}

func CmdRegister(backendState *BackendState, request common.RegisterBody) error {
	process_description := ProcessDescription{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		OutFilePath:           "/home/maciej/work/Spieven/test_scripts/log.txt",
		MaxSubsequentFailures: 3,
	}

	registered := backendState.processes.TryRegisterProcess(&process_description, &backendState.messages)
	if registered {
		backendState.messages.AddF(BackendMessageInfo, "Registered process %v", request.Cmdline)
	}
	return nil
}
