package backend

import (
	"net"
	"supervisor/common"
	"time"
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

// TODO rename register->schedule
func CmdRegister(backendState *BackendState, frontendConnection net.Conn, request common.RegisterBody) error {
	task := Task{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		OutFilePath:           "/home/maciej/work/Spieven/test_scripts/log.txt", // TODO define some directory for logs and generate proper paths
		MaxSubsequentFailures: 3,
		Env:                   request.Env,
		UserIndex:             request.UserIndex,
		FriendlyName:          request.FriendlyName,
	}

	var responseStatus byte
	switch TryScheduleTask(&task, backendState) {
	case ScheduleResultSuccess:
		backendState.messages.Add(BackendMessageInfo, &task, "Scheduled task")
		responseStatus = common.RegisterResponseSuccess
	case ScheduleResultAlreadyRunning:
		backendState.messages.Add(BackendMessageError, nil, "Task already running")
		responseStatus = common.RegisterResponseAlreadyRunning
	case ScheduleResultInvalidDisplay:
		backendState.messages.Add(BackendMessageError, nil, "Task uses invalid display")
		responseStatus = common.RegisterResponseAlreadyRunning
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(BackendMessageError, nil, "Unknown scheduling error")
		responseStatus = common.RegisterResponseUnknown
	}

	response := common.RegisterResponseBody{
		Id:      task.Computed.Id,
		Status:  responseStatus,
		LogFile: task.OutFilePath,
	}

	packet, err := common.EncodeRegisterResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}

func CmdNotifyTaskEnd(backendState *BackendState, frontendConnection net.Conn, taskId int) error {
	scheduler := &backendState.scheduler

	var response common.NotifyTaskEndResponseBody
	var reponseSet bool

	for !reponseSet {
		// TODO ensure this loop exits when frontend exits early

		scheduler.lock.Lock()

		if taskId > scheduler.currentId {
			response = common.NotifyTaskEndResponseInvalidTask
			reponseSet = true
		}

		stillRunning := false
		for _, task := range scheduler.tasks {
			if task.Computed.Id == taskId && !task.Dynamic.IsDeactivated {
				stillRunning = true
			}
		}

		if !stillRunning {
			response = common.NotifyTaskEndResponseEnded
			reponseSet = true
		}

		scheduler.lock.Unlock()

		time.Sleep(time.Millisecond * 500)
	}

	packet, err := common.EncodeNotifyTaskEndResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}
