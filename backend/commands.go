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
		responseItem.OutFilePath = task.Computed.OutFilePath
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

func CmdSchedule(backendState *BackendState, frontendConnection net.Conn, request common.ScheduleBody) error {
	task := Task{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		MaxSubsequentFailures: 3,
		Env:                   request.Env,
		UserIndex:             request.UserIndex,
		FriendlyName:          request.FriendlyName,
	}

	var responseStatus byte
	switch TryScheduleTask(&task, backendState) {
	case ScheduleResultSuccess:
		backendState.messages.Add(BackendMessageInfo, &task, "Scheduled task")
		responseStatus = common.ScheduleResponseSuccess
	case ScheduleResultAlreadyRunning:
		backendState.messages.Add(BackendMessageError, nil, "Task already running")
		responseStatus = common.ScheduleResponseAlreadyRunning
	case ScheduleResultInvalidDisplay:
		backendState.messages.Add(BackendMessageError, nil, "Task uses invalid display")
		responseStatus = common.ScheduleResponseAlreadyRunning
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(BackendMessageError, nil, "Unknown scheduling error")
		responseStatus = common.ScheduleResponseUnknown
	}

	response := common.ScheduleResponseBody{
		Id:      task.Computed.Id,
		Status:  responseStatus,
		LogFile: task.Computed.OutFilePath,
	}

	packet, err := common.EncodeScheduleResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}

func CmdQueryTaskActive(backendState *BackendState, frontendConnection net.Conn, taskId int) error {
	scheduler := &backendState.scheduler

	var response common.QueryTaskActiveResponseBody

	scheduler.lock.Lock()
	if taskId < scheduler.currentId {
		response = common.QueryTaskActiveResponseBodyInactive
		for _, task := range scheduler.tasks {
			if task.Computed.Id == taskId && !task.Dynamic.IsDeactivated {
				response = common.QueryTaskActiveResponseBodyActive
			}
		}
	} else {
		response = common.QueryTaskActiveResponseInvalidTask
	}
	scheduler.lock.Unlock()

	packet, err := common.EncodeQueryTaskActiveResponsePacket(response)
	if err != nil {
		return err
	}

	return common.SendPacket(frontendConnection, packet)
}
