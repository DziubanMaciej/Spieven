package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"supervisor/common"
	"supervisor/common/types"
	"sync"
	"time"
)

type Scheduler struct {
	tasks     []*Task
	currentId int
	lock      common.CheckedLock

	_ common.NoCopy
}

func (scheduler *Scheduler) Trim(backendMessages *BackendMessages, files *FilePathProvider) {
	scheduler.lock.AssertLocked()

	var tasksToKeep []*Task
	var tasksToDeactivate []*Task

	// Divide tasks we have in memory into still active and deactivated tasks
	for _, currTask := range scheduler.tasks {
		if currTask.Dynamic.IsDeactivated {
			tasksToDeactivate = append(tasksToDeactivate, currTask)
		} else {
			tasksToKeep = append(tasksToKeep, currTask)
		}
	}

	// Push deactivated tasks out to a file
	if len(tasksToDeactivate) > 0 {
		filePath := files.GetDeactivatedTasksFile()
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			backendMessages.AddF(BackendMessageError, nil, "Failed to open %s. Cannot push deactivated tasks out of memory to a file.", filePath)
			tasksToKeep = scheduler.tasks // Keep all tasks, so we don't lose data
		} else {
			for _, currTask := range tasksToDeactivate {
				// We're saving this as ndjson - json objects delimeted by newlines. For obvious reasons no fields of
				// Task can contain newlines. User inputs are sanitized for newlines, so they shouldn't appear in the
				// task struct.
				serializedTask, err := json.Marshal(currTask)
				serializedTask = append(serializedTask, '\n')

				if err == nil {
					err = common.WriteBytesToWriter(file, serializedTask)
				}

				if err == nil {
					backendMessages.Add(BackendMessageInfo, currTask, "Trimmed task")
				} else {
					backendMessages.AddF(BackendMessageError, currTask, "Failed to trim task: %s", err)
					tasksToKeep = append(tasksToKeep, currTask)
				}
			}
			file.Close()
		}

	}

	// Keep still active tasks in memory
	scheduler.tasks = tasksToKeep
}

func (scheduler *Scheduler) ReadTrimmedTasks(backendMessages *BackendMessages, files *FilePathProvider) []*Task {
	scheduler.lock.AssertLocked()

	var result []*Task

	filePath := files.GetDeactivatedTasksFile()
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		backendMessages.AddF(BackendMessageError, nil, "Failed reading trimmed tasks: %s", err.Error())
		return result
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var task Task
		err := json.Unmarshal(scanner.Bytes(), &task)
		if err != nil {
			backendMessages.AddF(BackendMessageError, nil, "Failed decoding a task from %s: %s", filePath, err.Error())
			continue
		}

		result = append(result, &task)
	}

	return result
}

func (scheduler *Scheduler) ExtractDeactivatedTask(taskId int, backendState *BackendState) (*Task, types.ScheduleResponseStatus) {
	scheduler.lock.AssertLocked()

	backendMessages := backendState.messages

	var extractedTask *Task

	// First look in memory
	{
		var indexToRemove int
		for index, currTask := range scheduler.tasks {
			if currTask.Computed.Id == taskId {
				if currTask.Dynamic.IsDeactivated {
					indexToRemove = index
					extractedTask = currTask
					break
				} else {
					return nil, types.ScheduleResponseStatusTaskNotDeactivated
				}
			}
		}

		if extractedTask != nil {
			// Remove task from the list and return it
			newCount := len(scheduler.tasks) - 1
			scheduler.tasks[indexToRemove] = scheduler.tasks[newCount]
			scheduler.tasks = scheduler.tasks[:newCount]
			return extractedTask, types.ScheduleResponseStatusSuccess
		}

	}

	// If we're here, we didn't find a task in memory. Look in ndjson file. We'll have to remove extracted task from the file,
	// so we're also opening temporary output file and writing to it all lines except for the removed task. Then we'll copy
	// it to be the new ndjson file.
	{
		inputFilePath := backendState.files.GetDeactivatedTasksFile()
		inputFile, err := os.OpenFile(inputFilePath, os.O_RDONLY, 0644)
		if err != nil {
			backendMessages.AddF(BackendMessageError, nil, "Failed reading trimmed tasks: %s", err.Error())
			return nil, types.ScheduleResponseStatusTaskNotFound
		}
		defer inputFile.Close()

		outputFile, err := backendState.files.GetTmpFile()
		outputFilePath := outputFile.Name()
		defer os.Remove(outputFilePath)
		defer outputFile.Close()
		if err != nil {
			backendMessages.AddF(BackendMessageError, nil, "Failed opening tmp file: %s", err.Error())
			return nil, types.ScheduleResponseStatusTaskNotFound
		}

		scanner := bufio.NewScanner(inputFile)
		for scanner.Scan() {
			line := scanner.Bytes()
			var currentTask Task
			err := json.Unmarshal(line, &currentTask)
			if err != nil {
				backendMessages.AddF(BackendMessageError, nil, "Failed decoding a task from %s: %s", inputFilePath, err.Error())
				continue
			}

			if extractedTask == nil && currentTask.Computed.Id == taskId {
				extractedTask = &currentTask
			} else {
				line = append(line, '\n')
				if err := common.WriteBytesToWriter(outputFile, line); err != nil {
					backendMessages.AddF(BackendMessageError, nil, "Failed writing to tmp file")
					return nil, types.ScheduleResponseStatusTaskNotFound
				}
			}
		}

		if extractedTask != nil {
			inputFile.Close()
			outputFile.Close()
			if err := common.CopyFile(outputFilePath, inputFilePath); err != nil {
				backendMessages.AddF(BackendMessageError, nil, "Failed copying tmp file to ndjson")
				return nil, types.ScheduleResponseStatusTaskNotFound
			}

			return extractedTask, types.ScheduleResponseStatusSuccess
		}
	}

	// If we're here, we didn't find the task neither in memory nor in ndjson file
	return nil, types.ScheduleResponseStatusTaskNotFound
}

func (scheduler *Scheduler) CheckForTaskConflict(newTask *Task) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	for _, currTask := range scheduler.tasks {
		if !currTask.Dynamic.IsDeactivated {
			if currTask.Computed.Hash == newTask.Computed.Hash {
				return types.ScheduleResponseStatusAlreadyRunning
			}
			if currTask.FriendlyName != "" && currTask.Computed.NameDisplayHash == newTask.Computed.NameDisplayHash {
				return types.ScheduleResponseStatusNameDisplayAlreadyRunning
			}
		}
	}

	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) CheckForDisplay(newTask *Task, backendState *BackendState) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	switch newTask.Display.Type {
	case types.DisplaySelectionTypeHeadless:
	case types.DisplaySelectionTypeXorg:
		_, err := GetXorgDisplay(newTask.Display.Name, backendState)
		if err != nil {
			return types.ScheduleResponseStatusInvalidDisplay
		}
	case types.DisplaySelectionTypeWayland:
		// TODO implement wayland detection
		backendState.messages.Add(BackendMessageError, newTask, "Wayland display tracking is not implemented")
	default:
		backendState.messages.Add(BackendMessageError, newTask, "Invalid display type")
	}

	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) TryScheduleTask(newTask *Task, backendState *BackendState) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	// Calculate internal properties
	newTask.Init(scheduler.currentId, backendState.files.GetTaskLogFile(scheduler.currentId))
	scheduler.currentId++

	// Do not schedule, if a similar task is already running
	if status := scheduler.CheckForTaskConflict(newTask); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Ensure display is correct
	if status := scheduler.CheckForDisplay(newTask, backendState); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Schedule
	scheduler.tasks = append(scheduler.tasks, newTask)
	backendState.StartGoroutine(func() {
		ExecuteTask(newTask, backendState)
	})
	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) TryRescheduleTask(newTask *Task, backendState *BackendState) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	// Calculate internal properties
	newTask.Init(newTask.Computed.Id, backendState.files.GetTaskLogFile(newTask.Computed.Id))

	// Do not schedule, if a similar task is already running
	if status := scheduler.CheckForTaskConflict(newTask); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Ensure display is correct
	if status := scheduler.CheckForDisplay(newTask, backendState); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Schedule
	scheduler.tasks = append(scheduler.tasks, newTask)
	backendState.StartGoroutine(func() {
		ExecuteTask(newTask, backendState)
	})
	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) StopTasksByDisplay(displayType types.DisplaySelectionType, displayName string) {
	scheduler.lock.AssertLocked()

	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	for _, currTask := range scheduler.tasks {
		if currTask.Display.Name == displayName && currTask.Display.Type == displayType {
			currTask.Channels.StopChannel <- fmt.Sprintf("stopping tasks on display %v", displayName)
		}
	}
}

func ExecuteTask(task *Task, backendState *BackendState) {
	// Initialize per-task logger
	perTaskLogger := CreateFileLogger(backendState, task.Computed.Id, task.CaptureStdout)
	err := perTaskLogger.run()
	if err != nil {
		backendState.messages.Add(BackendMessageError, task, "failed to create per-task logger")
		return
	}
	defer perTaskLogger.stop()

	// Copy the dynamic portion of task structure. Updates to it must be synchronized. We will be updating a local
	// copy and assign it to the actual task struct under a lock in one go every time something changes. Technically
	// this initial copy doesn't need a lock, because no other routine than ExecuteTask should ever change task.Dynamic.
	// But, for completeness we're still locking.
	backendState.scheduler.lock.Lock()
	shadowDynamicState := task.Dynamic
	backendState.scheduler.lock.Unlock()

	// Logging in this function is a bit complicated. We have 3 possible places where logs can go:
	//  1. FileLogger - per-task file with detailed info about the current task as well as stdout/stderr. All messages
	//    will go there.
	//  2. BackendMessages - messages global for the entire Spieven backend. This is more corse-grained. Only some
	//    message will go there, not to bloat the log.
	//  3. Deactivation reason - when this task becomes deactivate, we'll store a reason why. This will be saved within
	//    the task struct and frontend will be able to retrieve it. Only one log can be used as a deactivation reason.
	// In order to simplify handling all possible behaviors, use the wrapper functions below.
	type LogFlag uint
	const (
		LogTask                      = 1
		LogBackend                   = 2
		LogDeactivation              = 4
		LogFlagErr           LogFlag = 8
		LogFlagTaskSeparator         = 16
	)
	log := func(flags LogFlag, content string) {
		hasFlag := func(f LogFlag) bool {
			return (flags & f) != 0
		}

		if hasFlag(LogDeactivation) {
			content += " Deactivating."

			shadowDynamicState.IsDeactivated = true
			shadowDynamicState.DeactivatedReason = content
		}
		if hasFlag(LogDeactivation | LogBackend) {
			severity := BackendMessageInfo
			if hasFlag(LogFlagErr) {
				severity = BackendMessageError
			}
			backendState.messages.Add(severity, task, content)
		}
		if hasFlag(LogDeactivation | LogBackend | LogTask) {
			perTaskLogger.channel <- diagnosticMessage(content, hasFlag(LogFlagTaskSeparator))
		}

	}
	logF := func(flags LogFlag, format string, args ...any) {
		content := fmt.Sprintf(format, args...)
		log(flags, content)
	}

	// Write LogTask with general info about the task
	logF(LogTask, "Task information:")
	logF(LogTask, "  Id: %v", task.Computed.Id)
	logF(LogTask, "  FriendlyName: %v", task.FriendlyName)
	logF(LogTask, "  Cmdline: %v", task.Cmdline)
	logF(LogTask, "  Cwd: %v", task.Cwd)
	logF(LogTask, "  DisplayType=%v DisplayName=%v", task.Display.Type, task.Display.Name)

	// Execute the main loop until the task becomes deactivated.
	for !shadowDynamicState.IsDeactivated {
		// Initialize the command struct
		cmdContext, cmdCancel := context.WithCancel(backendState.context)
		defer cmdCancel()
		cmd := exec.CommandContext(cmdContext, task.Cmdline[0], task.Cmdline[1:]...)
		cmd.Dir = task.Cwd
		cmd.Env = task.Env
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to create stdout pipe.")
			break
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to create stderr pipe.")
			break
		}

		// Start the command
		err = cmd.Start()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to start the command.")
			break
		}
		log(LogTask, "Command started.")

		// Run pipe reading goroutines
		var pipeWaitGroup sync.WaitGroup
		pipeWaitGroup.Add(2)
		backendState.StartGoroutine(func() {
			perTaskLogger.streamOutput(stdoutPipe)
			pipeWaitGroup.Done()
		})
		backendState.StartGoroutine(func() {
			perTaskLogger.streamOutput(stderrPipe)
			pipeWaitGroup.Done()
		})

		// Wait for the command in a separate goroutine and signal when it ends. It's important to first wait for the
		// goroutines streaming the output. Otherwise, cmd.Wait() will close the pipes leading to a race condition.
		commandResultChannel := make(chan int, 1)
		backendState.StartGoroutine(func() {
			pipeWaitGroup.Wait()

			status := 0
			err := cmd.Wait()

			if err != nil {
				status = err.(*exec.ExitError).ExitCode()
			}
			commandResultChannel <- status
		})

		// Block until something happens
		commandSuccess := false
		select {
		case <-cmdContext.Done():
			// cmdContext derives from BackendState's context, which is killed by Ctrl+C interrupt
			logF(LogTask|LogDeactivation, "Backend killed.")
		case exitCode := <-commandResultChannel:
			// Command ended normally
			logF(LogTask, "Command ended with code %v.", exitCode)
			shadowDynamicState.LastExitValue = exitCode
			if exitCode == 0 {
				commandSuccess = true
			}
		case <-perTaskLogger.errorChannel:
			// Logger failed. We don't want to execute the command without logging. Kill it and return error.
			log(LogDeactivation|LogFlagErr, "Failed logging.")
		case reason := <-task.Channels.StopChannel:
			logF(LogDeactivation, "Task killed (%v).", reason)
		}

		// Send a separator to the per-task logger and wait for its response via channel. If it's valid, assign
		// it the task's dynamic state.
		{
			log(LogTask|LogFlagTaskSeparator, "")
			stdoutPath := <-perTaskLogger.stdoutFilePathChannel

			if stdoutPath != "" && !common.FileExists(stdoutPath) {
				logF(LogBackend|LogFlagErr, "Incorrect stdout file path from per-task logger: %v", stdoutPath)
				stdoutPath = ""
			}

			shadowDynamicState.LastStdoutFilePath = stdoutPath
		}

		// Update execution and failure counts
		shadowDynamicState.RunCount++
		if commandSuccess {
			shadowDynamicState.FailureCount = 0
		} else {
			shadowDynamicState.FailureCount++
			shadowDynamicState.SubsequentFailureCount++
		}

		// Handle MaxSubsequentFailures
		if task.MaxSubsequentFailures >= 0 && shadowDynamicState.SubsequentFailureCount >= task.MaxSubsequentFailures {
			logF(LogDeactivation, "Task reached subsequent failure count limit of %v.", task.MaxSubsequentFailures)
		}

		// Update dynamic state
		backendState.scheduler.lock.Lock()
		task.Dynamic = shadowDynamicState
		backendState.scheduler.lock.Unlock()

		// Perform delay between command executions
		if !shadowDynamicState.IsDeactivated {
			delay := task.DelayAfterFailureMs
			if commandSuccess {
				delay = task.DelayAfterSuccessMs
			}
			InterruptibleSleep(time.Millisecond*time.Duration(delay), task.Channels.RefreshChannel)
		}
	}

	// Update dynamic state in case we broke from the loop
	backendState.scheduler.lock.Lock()
	task.Dynamic = shadowDynamicState
	backendState.scheduler.lock.Unlock()
}

func InterruptibleSleep(d time.Duration, refreshChannel chan struct{}) {
	// Start a timer
	timer := time.NewTimer(d)
	defer timer.Stop()

	// Wait for either the timer or the stop channel
	select {
	case <-timer.C:
	case <-refreshChannel:
	}
}
