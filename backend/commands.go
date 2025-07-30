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
	messages := backendState.messages

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

func CmdList(backendState *BackendState, frontendConnection net.Conn, request common.ListBody) error {
	scheduler := &backendState.scheduler

	scheduler.lock.Lock()
	response := make(common.ListResponseBody, 0)
	appendTask := func(task *Task) {
		item := common.ListResponseBodyItem{
			Id:                    task.Computed.Id,
			Cmdline:               task.Cmdline,
			Cwd:                   task.Cwd,
			OutFilePath:           task.Computed.OutFilePath,
			MaxSubsequentFailures: task.MaxSubsequentFailures,
			UserIndex:             task.UserIndex,
			IsDeactivated:         task.Dynamic.IsDeactivated,
			DeactivationReason:    task.Dynamic.DeactivatedReason,
			FriendlyName:          task.FriendlyName,
		}
		response = append(response, item)
	}

	for _, task := range scheduler.tasks {
		if !task.Dynamic.IsDeactivated || request.IncludeDeactivated {
			appendTask(task)
		}
	}

	if request.IncludeDeactivated {
		tasks := scheduler.ReadTrimmedTasks(backendState.messages, backendState.files)
		for _, task := range tasks {
			appendTask(task)
		}
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
