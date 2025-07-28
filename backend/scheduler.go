package backend

import (
	"context"
	"fmt"
	"os/exec"
	"supervisor/common"
	"sync"
	"time"
)

type Scheduler struct {
	tasks     []*Task
	currentId int
	lock      sync.Mutex

	_ common.NoCopy
}

func (scheduler *Scheduler) Trim(maxAge time.Duration, backendMessages *BackendMessages) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	now := time.Now()
	var newTasks []*Task

	for _, currTask := range scheduler.tasks {
		deadline := currTask.Dynamic.DeactivatedTime.Add(maxAge)
		if currTask.Dynamic.IsDeactivated && deadline.Before(now) {
			backendMessages.Add(BackendMessageInfo, currTask, "Trimmed task")
		} else {
			newTasks = append(newTasks, currTask)
		}
	}

	scheduler.tasks = newTasks
}

type ScheduleResult byte

const (
	ScheduleResultSuccess ScheduleResult = iota
	ScheduleResultAlreadyRunning
	ScheduleResultInvalidDisplay
)

func TryScheduleTask(newTask *Task, backendState *BackendState) ScheduleResult {
	scheduler := &backendState.scheduler

	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	// Calculate internal properties including the task's hash. Skip scheduling if we already have it.
	newTask.Init(scheduler.currentId)
	for _, currTask := range scheduler.tasks {
		if !currTask.Dynamic.IsDeactivated && currTask.Computed.Hash == newTask.Computed.Hash {
			return ScheduleResultAlreadyRunning
		}
	}

	// Ensure display is correct
	switch newTask.Computed.DisplayType {
	case DisplayNone:
	case DisplayXorg:
		_, err := backendState.displays.GetXorgDisplay(newTask.Computed.DisplayName, scheduler)
		if err != nil {
			return ScheduleResultInvalidDisplay
		}
	default:
		panic("Not implemented")
	}

	// Increment id assigned for new tasks.
	scheduler.currentId++

	scheduler.tasks = append(scheduler.tasks, newTask)
	go ExecuteTask(newTask, backendState)
	return ScheduleResultSuccess
}

func (scheduler *Scheduler) StopTasksByDisplay(displayType DisplayType, displayName string) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	for _, currTask := range scheduler.tasks {
		if currTask.Computed.DisplayName == displayName && currTask.Computed.DisplayType == displayType {
			currTask.Channels.StopChannel <- fmt.Sprintf("stopping tasks on display %v", displayName)
		}
	}
}

func ExecuteTask(task *Task, backendState *BackendState) {
	// Initialize per-task logger
	perTaskLogger := CreateFileLogger(task.OutFilePath)
	err := perTaskLogger.run()
	if err != nil {
		backendState.messages.Add(BackendMessageError, task, "failed to create per-task logger")
		return
	}
	defer perTaskLogger.stop()

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
	isTaskDeactivated := false
	log := func(flags LogFlag, content string) {
		hasFlag := func(f LogFlag) bool {
			return (flags & f) != 0
		}

		if hasFlag(LogDeactivation) {
			content += " Deactivating."

			isTaskDeactivated = true
			task.Deactivate(content) // TODO this is not synchronized!
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
	logF(LogTask, "  UserIndex: %v", task.UserIndex)
	logF(LogTask, "  Cmdline: %v", task.Cmdline)
	logF(LogTask, "  Cwd: %v", task.Cwd)
	logF(LogTask|LogFlagTaskSeparator, "  DisplayType=%v DisplayName=%v", task.Computed.DisplayType, task.Computed.DisplayName)

	// Execute the main loop until the task becomes deactivated.
	subsequentFailures := 0
	for !isTaskDeactivated {
		// Initialize the command struct
		cmdContext, cmdCancel := context.WithCancel(context.Background())
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
			log(LogDeactivation|LogFlagErr, "Failed to create stdout pipe.")
			break
		}
		log(LogTask, "Command started.")

		// Run pipe reading goroutines
		var pipeWaitGroup sync.WaitGroup
		pipeWaitGroup.Add(2)
		go func() {
			perTaskLogger.streamOutput(stdoutPipe)
			pipeWaitGroup.Done()
		}()
		go func() {
			perTaskLogger.streamOutput(stderrPipe)
			pipeWaitGroup.Done()
		}()

		// Wait for the command in a separate goroutine and signal when it ends. It's important to first wait for the
		// goroutines streaming the output. Otherwise, cmd.Wait() will close the pipes leading to a race condition.
		commandResultChannel := make(chan int)
		go func() {
			pipeWaitGroup.Wait()

			status := 0
			err := cmd.Wait()

			if err != nil {
				status = err.(*exec.ExitError).ExitCode()
			}
			commandResultChannel <- status
		}()

		// Block until something happens
		select {
		case exitCode := <-commandResultChannel:
			// Command ended.
			logF(LogTask|LogFlagTaskSeparator, "Command ended with code %v.", exitCode)
			if exitCode == 0 {
				subsequentFailures = 0
			} else {
				subsequentFailures++
			}
		case <-perTaskLogger.errorChannel:
			// Logger failed. We don't want to execute the command without logging. Kill it and return error.
			log(LogDeactivation|LogFlagErr, "Failed logging.")
		case reason := <-task.Channels.StopChannel:
			logF(LogDeactivation, "Command killed (%v).", reason)
		}

		// Handle breaks from the main loop
		if task.MaxSubsequentFailures >= 0 && subsequentFailures >= task.MaxSubsequentFailures {
			logF(LogDeactivation, "Task reached subsequent failure count limit of %v.", task.MaxSubsequentFailures)
			break
		}
	}
}
