package backend

import (
	"net"
	"supervisor/common/packet"
	"supervisor/common/types"
)

func CmdLog(backendState *BackendState, frontendConnection net.Conn) error {
	messages := backendState.messages

	messages.lock.Lock()
	response := make(packet.LogResponseBody, len(messages.messages))
	for i, message := range messages.messages {
		response[i] = message.String()
	}
	messages.lock.Unlock()

	reponsePacket, err := packet.EncodeLogResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, reponsePacket)
}

func CmdList(backendState *BackendState, frontendConnection net.Conn, request packet.ListRequestBody) error {
	scheduler := &backendState.scheduler

	response := make(packet.ListResponseBody, 0)
	appendTask := func(task *Task) {
		stdout, err := task.ReadLastStdout()
		hasStdout := true
		if err != nil {
			hasStdout = false
		}

		item := packet.ListResponseBodyItem{
			Id:                     task.Computed.Id,
			Cmdline:                task.Cmdline,
			Cwd:                    task.Cwd,
			Display:                task.Display,
			OutFilePath:            task.Computed.OutFilePath,
			IsDeactivated:          task.Dynamic.IsDeactivated,
			DeactivationReason:     task.Dynamic.DeactivatedReason,
			FriendlyName:           task.FriendlyName,
			RunCount:               task.Dynamic.RunCount,
			FailureCount:           task.Dynamic.FailureCount,
			SubsequentFailureCount: task.Dynamic.SubsequentFailureCount,
			MaxSubsequentFailures:  task.MaxSubsequentFailures,
			LastExitValue:          task.Dynamic.LastExitValue,
			LastStdout:             stdout,
			HasLastStdout:          hasStdout,
		}
		response = append(response, item)
	}

	scheduler.lock.Lock()

	// Prepare a selector function, that returns true when a task should be sent back to the frontend.
	// By default we want to return all of them, but then we compose additional checks depending on
	// the frontend request.
	selector := func(task *Task) bool { return true }
	if !request.IncludeDeactivated {
		prev := selector
		selector = func(task *Task) bool { return prev(task) && !task.Dynamic.IsDeactivated }
	}
	request.Filter.Derive()
	if request.Filter.HasIdFilter {
		prev := selector
		selector = func(task *Task) bool { return prev(task) && task.Computed.Id == request.Filter.IdFilter }
	}
	if request.Filter.HasNameFilter {
		prev := selector
		selector = func(task *Task) bool { return prev(task) && task.FriendlyName == request.Filter.NameFilter }
	}
	if request.Filter.HasDisplayFilter {
		prev := selector
		selector = func(task *Task) bool { return prev(task) && task.Display == request.Filter.DisplayFilter }
	}

	// First look through in-memory list of tasks. Some of them will be active, some can be deactivated,
	// depending on when Trim() was called.
	for _, task := range scheduler.tasks {
		if selector(task) {
			appendTask(task)
		}
	}

	// If we're interested in deactivated tasks, load them from a file. The selector invocation could be moved into
	// scheduler to avoid building list of all tasks, but let's not worry about that now.
	if request.IncludeDeactivated {
		tasks := scheduler.ReadTrimmedTasks(backendState.messages, backendState.files)
		for _, task := range tasks {
			if selector(task) {
				appendTask(task)
			}
		}
	}

	scheduler.lock.Unlock()

	reponsePacket, err := packet.EncodeListResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, reponsePacket)
}

func CmdSchedule(backendState *BackendState, frontendConnection net.Conn, request packet.ScheduleRequestBody) error {
	task := Task{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		DelayAfterSuccessMs:   request.DelayAfterSuccessMs,
		DelayAfterFailureMs:   request.DelayAfterFailureMs,
		MaxSubsequentFailures: request.MaxSubsequentFailures,
		Env:                   request.Env,
		FriendlyName:          request.FriendlyName,
		CaptureStdout:         request.CaptureStdout,
		Display:               request.Display,
	}

	responseStatus := TryScheduleTask(&task, backendState)
	response := packet.ScheduleResponseBody{
		Id:      task.Computed.Id,
		Status:  responseStatus,
		LogFile: task.Computed.OutFilePath,
	}

	switch responseStatus {
	case types.ScheduleResponseStatusSuccess:
		backendState.messages.Add(BackendMessageInfo, &task, "Scheduled task")
	case types.ScheduleResponseStatusAlreadyRunning:
		backendState.messages.Add(BackendMessageError, nil, "Task already running")
	case types.ScheduleResponseStatusNameDisplayAlreadyRunning:
		backendState.messages.AddF(BackendMessageError, nil, "Task named %v already present on \"%v\" display", task.FriendlyName, task.Display.ComputeDisplayLabel())
	case types.ScheduleResponseStatusInvalidDisplay:
		backendState.messages.Add(BackendMessageError, nil, "Task uses invalid display")
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(BackendMessageError, nil, "Unknown scheduling error")
		response.Status = types.ScheduleResponseStatusUnknown
	}

	responsePacket, err := packet.EncodeScheduleResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}

func CmdQueryTaskActive(backendState *BackendState, frontendConnection net.Conn, request packet.QueryTaskActiveRequestBody) error {
	scheduler := &backendState.scheduler
	taskId := int(request)

	var response packet.QueryTaskActiveResponseBody

	scheduler.lock.Lock()
	if taskId < scheduler.currentId {
		response = packet.QueryTaskActiveResponseBodyInactive
		for _, task := range scheduler.tasks {
			if task.Computed.Id == taskId && !task.Dynamic.IsDeactivated {
				response = packet.QueryTaskActiveResponseBodyActive
			}
		}
	} else {
		response = packet.QueryTaskActiveResponseInvalidTask
	}
	scheduler.lock.Unlock()

	responsePacket, err := packet.EncodeQueryTaskActiveResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}
