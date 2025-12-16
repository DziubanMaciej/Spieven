package backend

import (
	"net"
	i "spieven/backend/interfaces"
	"spieven/backend/scheduler"
	"spieven/common"
	"spieven/common/packet"
	"spieven/common/types"
)

func getSelectorFunc(filter *types.TaskFilter) func(*scheduler.Task) bool {
	filter.Derive()

	// By default we want to return all of them, but then we compose additional checks depending on
	// the frontend request.
	selector := func(task *scheduler.Task) bool { return true }
	if filter.HasIdFilter {
		prev := selector
		selector = func(task *scheduler.Task) bool { return prev(task) && task.Computed.Id == filter.IdFilter }
	}
	if filter.HasAnyNameFilter {
		prev := selector
		selector = func(task *scheduler.Task) bool {
			return prev(task) && common.Contains(filter.AnyNameFilter, task.FriendlyName)
		}
	}
	if filter.HasDisplayFilter {
		prev := selector
		selector = func(task *scheduler.Task) bool { return prev(task) && task.Display == filter.DisplayFilter }
	}
	if filter.HasAllTagsFilter {
		prev := selector
		selector = func(task *scheduler.Task) bool {
			return prev(task) && common.ContainsAll(filter.AllTagsFilter, task.Tags)
		}
	}
	return selector
}

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
	sched := &backendState.scheduler

	namesMap := make(map[string][]int) // this map store a list of indices of task for each friendlyName
	response := make(packet.ListResponseBody, 0)

	appendTask := func(task *scheduler.Task) {
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
			Tags:                   task.Tags,
			RunCount:               task.Dynamic.RunCount,
			FailureCount:           task.Dynamic.FailureCount,
			SubsequentFailureCount: task.Dynamic.SubsequentFailureCount,
			MaxSubsequentFailures:  task.MaxSubsequentFailures,
			LastExitValue:          task.Dynamic.LastExitValue,
			LastStdout:             stdout,
			HasLastStdout:          hasStdout,
		}

		if request.UniqueNames {
			namesMap[item.FriendlyName] = append(namesMap[item.FriendlyName], len(response))
		}

		response = append(response, item)
	}

	sched.Lock()

	// Prepare a selector function, that returns true when a task should be sent back to the frontend.
	selector := getSelectorFunc(&request.Filter)

	// Prepare helper functions to list tasks from memory or from disc. The in-memory list can contain both active tasks
	// and already deactivated tasks that were not yet paged to a file.
	getTasksFromMemory := func(allowDeactivated bool) {
		for _, task := range sched.GetTasks() {
			if selector(task) && (allowDeactivated || !task.Dynamic.IsDeactivated) {
				appendTask(task)
			}
		}
	}
	getTasksFromDeactivatedFile := func() {
		tasks := sched.ReadTrimmedTasks(backendState.messages, backendState.files)
		for _, task := range tasks {
			if selector(task) {
				appendTask(task)
			}
		}
	}

	// Retrieve tasks based on ActiveOnly flag. If we need only active tasks, we only have to look at in memory tasks. Otherwise,
	// we also include deactivated tasks that are either still in memory or were paged out to a file.
	if request.ActiveOnly {
		getTasksFromMemory(false)
	} else {
		getTasksFromMemory(true)
		getTasksFromDeactivatedFile()
	}

	sched.Unlock()

	// If unique names were requested, look through the map and for each name that has multiple tasks select the one with
	// the highest id. Remove all others.
	if request.UniqueNames {
		newResponse := make(packet.ListResponseBody, 0)
		for _, indices := range namesMap {
			selectedItem := response[indices[0]]

			for _, taskIndex := range indices {
				currentItem := response[taskIndex]
				if currentItem.Id > selectedItem.Id {
					selectedItem = currentItem
				}
			}

			newResponse = append(newResponse, selectedItem)
		}

		response = newResponse
	}

	reponsePacket, err := packet.EncodeListResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, reponsePacket)
}

func CmdSchedule(backendState *BackendState, frontendConnection net.Conn, request packet.ScheduleRequestBody) error {
	sched := &backendState.scheduler

	task := scheduler.Task{
		Cmdline:               request.Cmdline,
		Cwd:                   request.Cwd,
		DelayAfterSuccessMs:   request.DelayAfterSuccessMs,
		DelayAfterFailureMs:   request.DelayAfterFailureMs,
		MaxSubsequentFailures: request.MaxSubsequentFailures,
		Env:                   request.Env,
		FriendlyName:          request.FriendlyName,
		CaptureStdout:         request.CaptureStdout,
		Display:               request.Display,
		Tags:                  request.Tags,
	}

	sched.Lock()
	responseStatus := sched.TryScheduleTask(&task, backendState.files, backendState.displays, backendState.sync, backendState.messages)
	sched.Unlock()

	response := packet.ScheduleResponseBody{
		Id:      task.Computed.Id,
		Status:  responseStatus,
		LogFile: task.Computed.OutFilePath,
	}

	switch responseStatus {
	case types.ScheduleResponseStatusSuccess:
		backendState.messages.Add(i.BackendMessageInfo, &task, "Scheduled task")
	case types.ScheduleResponseStatusAlreadyRunning:
		backendState.messages.Add(i.BackendMessageError, nil, "Task already running")
	case types.ScheduleResponseStatusNameDisplayAlreadyRunning:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task named %v already present on \"%v\" display", task.FriendlyName, task.Display.ComputeDisplayLabel())
	case types.ScheduleResponseStatusInvalidDisplay:
		backendState.messages.Add(i.BackendMessageError, nil, "Task uses invalid display")
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(i.BackendMessageError, nil, "Unknown scheduling error")
		response.Status = types.ScheduleResponseStatusUnknown
	}

	responsePacket, err := packet.EncodeScheduleResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}

func CmdQueryTaskActive(backendState *BackendState, frontendConnection net.Conn, request packet.QueryTaskActiveRequestBody) error {
	sched := &backendState.scheduler
	taskId := int(request)

	var response packet.QueryTaskActiveResponseBody

	sched.Lock()
	if sched.IsValidId(taskId) {
		response = packet.QueryTaskActiveResponseBodyInactive
		for _, task := range sched.GetTasks() {
			if task.Computed.Id == taskId && !task.Dynamic.IsDeactivated {
				response = packet.QueryTaskActiveResponseBodyActive
			}
		}
	} else {
		response = packet.QueryTaskActiveResponseInvalidTask
	}
	sched.Unlock()

	responsePacket, err := packet.EncodeQueryTaskActiveResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}

func CmdRefresh(backendState *BackendState, frontendConnection net.Conn, request packet.RefreshRequestBody) error {
	sched := &backendState.scheduler

	var response packet.RefreshResponseBody

	sched.Lock()

	selector := getSelectorFunc(&request.Filter)

	for _, task := range sched.GetTasks() {
		if selector(task) {
			select {
			case task.Channels.RefreshChannel <- struct{}{}:
			default:
			}
			response.RefreshedTasksCount++
		}
	}
	response.ActiveTasksCount = len(sched.GetTasks())

	sched.Unlock()

	responsePacket, err := packet.EncodeRefreshResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}

func CmdReschedule(backendState *BackendState, frontendConnection net.Conn, request packet.RescheduleRequestBody) error {
	sched := &backendState.scheduler

	var response packet.RescheduleResponseBody

	sched.Lock()

	task, status := sched.ExtractDeactivatedTask(request.TaskId, backendState.files, backendState.messages)
	if status == types.ScheduleResponseStatusSuccess {
		response.Status = sched.TryRescheduleTask(task, backendState.files, backendState.displays, backendState.sync, backendState.messages)
		response.LogFile = task.Computed.OutFilePath
		response.Id = task.Computed.Id
	} else {
		response.Status = status

	}

	sched.Unlock()

	switch response.Status {
	case types.ScheduleResponseStatusSuccess:
		backendState.messages.AddF(i.BackendMessageInfo, task, "Rescheduled task %v", request.TaskId)
	case types.ScheduleResponseStatusAlreadyRunning:
		backendState.messages.Add(i.BackendMessageError, nil, "Task already running")
	case types.ScheduleResponseStatusNameDisplayAlreadyRunning:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task named %v already present on \"%v\" display", task.FriendlyName, task.Display.ComputeDisplayLabel())
	case types.ScheduleResponseStatusInvalidDisplay:
		backendState.messages.Add(i.BackendMessageError, nil, "Task uses invalid display")
	case types.ScheduleResponseStatusTaskNotFound:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task %v not found", request.TaskId)
	case types.ScheduleResponseStatusTaskNotDeactivated:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task %v is active, cannot reschedule", request.TaskId)
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(i.BackendMessageError, nil, "Unknown rescheduling error")
		response.Status = types.ScheduleResponseStatusUnknown
	}

	responsePacket, err := packet.EncodeRescheduleResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}

func CmdStop(backendState *BackendState, frontendConnection net.Conn, request packet.StopRequestBody) error {
	sched := &backendState.scheduler

	var response packet.StopResponseBody
	var foundTask *scheduler.Task

	sched.Lock()

	// First check if task ID is valid
	if sched.IsValidId(request.TaskId) {
		// Look for the task in memory
		for _, task := range sched.GetTasks() {
			if task.Computed.Id == request.TaskId {
				foundTask = task
				break
			}
		}

		if foundTask != nil && !foundTask.Dynamic.IsDeactivated {
			select {
			case foundTask.Channels.StopChannel <- "manually stopped":
			default:
				// Channel is full, but that's okay - multiple stop signals wouldn't change anything
			}
			response.Status = types.StopResponseStatusSuccess
		} else {
			// Task deactivated - either paged out or still in memory. This means the task is already stopped.
			response.Status = types.StopResponseStatusAlreadyStopped
		}
	} else {
		response.Status = types.StopResponseStatusTaskNotFound
	}

	sched.Unlock()

	switch response.Status {
	case types.StopResponseStatusSuccess:
		backendState.messages.AddF(i.BackendMessageInfo, foundTask, "Stopped task %v", request.TaskId)
	case types.StopResponseStatusTaskNotFound:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task %v not found", request.TaskId)
	case types.StopResponseStatusAlreadyStopped:
		backendState.messages.AddF(i.BackendMessageError, nil, "Task %v is already stopped", request.TaskId)
	default:
		// Shouldn't happen, but let's handle it gracefully
		backendState.messages.Add(i.BackendMessageError, nil, "Unknown stop error")
		response.Status = types.StopResponseStatusUnknown
	}

	responsePacket, err := packet.EncodeStopResponsePacket(response)
	if err != nil {
		return err
	}

	return packet.SendPacket(frontendConnection, responsePacket)
}
