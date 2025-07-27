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

func CmdLog(backendState *BackendState, frontendConnection net.Conn) error {
	messages := &backendState.messages

	messages.lock.Lock()
	response := make(common.LogResponseBody, len(messages.messages))
	for i, message := range messages.messages {
		response[i] = message.String()
	}
	messages.lock.Unlock()

	packet, err := common.EncodeLogResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}

func CmdList(backendState *BackendState, frontendConnection net.Conn) error {
	scheduler := &backendState.scheduler

	scheduler.lock.Lock()
	response := make(common.ListResponseBody, len(scheduler.tasks))
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
		responseItem.FriendlyName = task.FriendlyName
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
		FriendlyName:          request.FriendlyName,
	}

	registered := TryScheduleTask(&task, backendState)
	if registered {
		backendState.messages.Add(BackendMessageInfo, &task, "Registered process")
	} else {
		backendState.messages.Add(BackendMessageInfo, nil, "Did not register process, because it's already running")
	}

	packet, err := common.EncodeRegisterResponsePacket(registered)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}
