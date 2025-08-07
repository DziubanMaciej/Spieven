package backend

import (
	"net"
	"supervisor/common"
)

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

	response := make(common.ListResponseBody, 0)
	appendTask := func(task *Task) {
		stdout, err := task.ReadLastStdout()
		hasStdout := true
		if err != nil {
			hasStdout = false
		}

		item := common.ListResponseBodyItem{
			Id:                     task.Computed.Id,
			Cmdline:                task.Cmdline,
			Cwd:                    task.Cwd,
			OutFilePath:            task.Computed.OutFilePath,
			UserIndex:              task.UserIndex,
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
	if request.Filter.HasXorgDisplayFilter {
		prev := selector
		selector = func(task *Task) bool {
			return prev(task) && task.Computed.DisplayType == DisplayXorg && task.Computed.DisplayName == request.Filter.XorgDisplayFilter
		}
	}
	if request.Filter.HasWaylandDisplayFilter {
		prev := selector
		selector = func(task *Task) bool {
			return prev(task) && task.Computed.DisplayType == DisplayWayland && task.Computed.DisplayName == request.Filter.WaylandDisplayFilter
		}
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
		CaptureStdout:         request.CaptureStdout,
	}

	var responseStatus byte
	switch TryScheduleTask(&task, backendState) {
	case ScheduleResultSuccess:
		backendState.messages.Add(BackendMessageInfo, &task, "Scheduled task")
		responseStatus = common.ScheduleResponseSuccess
	case ScheduleResultAlreadyRunning:
		backendState.messages.Add(BackendMessageError, nil, "Task already running")
		responseStatus = common.ScheduleResponseAlreadyRunning
	case ScheduleResultNameDisplayAlreadyRunning:
		backendState.messages.AddF(BackendMessageError, nil, "Task named %v already present on %v display", task.FriendlyName, task.ComputeDisplayLabel())
		responseStatus = common.ScheduleResponseNameDisplayAlreadyRunning
	case ScheduleResultInvalidDisplay:
		backendState.messages.Add(BackendMessageError, nil, "Task uses invalid display")
		responseStatus = common.ScheduleResponseInvalidDisplay
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
