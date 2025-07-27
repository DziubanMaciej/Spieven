package backend

import (
	"net"
	"supervisor/common"
)

func CmdSummary(backendState *BackendState, frontendConnection net.Conn) error {
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

func CmdList(backendState *BackendState, frontendConnection net.Conn) error {
	scheduler := &backendState.scheduler

	response := make(common.ListResponseBody, len(scheduler.tasks))

	scheduler.lock.Lock()
	for i, task := range scheduler.tasks {
		responseItem := &response[i]

		responseItem.Id = task.Computed.Id
		responseItem.Cmdline = task.Cmdline
		responseItem.Cwd = task.Cwd
		responseItem.OutFilePath = task.OutFilePath
		responseItem.MaxSubsequentFailures = task.MaxSubsequentFailures
		responseItem.UserIndex = task.UserIndex
		responseItem.IsDeactivated = task.Dynamic.IsDeactivated
		responseItem.DeactivationReason = task.Dynamic.DeactivatedReason
	}
	scheduler.lock.Unlock()

	packet, err := common.EncodeListResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}

func CmdRegister(backendState *BackendState, frontendConnection net.Conn, request common.RegisterBody) error {
	task := Task{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		OutFilePath:           "/home/maciej/work/Spieven/test_scripts/log.txt",
		MaxSubsequentFailures: 3,
		Env:                   request.Env,
		UserIndex:             request.UserIndex,
	}

	registered := TryScheduleTask(&task, backendState)
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
