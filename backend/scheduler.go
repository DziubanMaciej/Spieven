package backend

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type Scheduler struct {
	tasks     []*Task
	currentId int
	lock      sync.Mutex
}

func TryScheduleTask(newTask *Task, backendState *BackendState) bool {
	scheduler := &backendState.scheduler

	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	// Calculate internal properties including the task's hash. Skip scheduling if we already have it.
	newTask.Init()
	for _, currTask := range scheduler.tasks {
		if currTask.Computed.Hash == newTask.Computed.Hash {
			return false
		}
	}

	// Ensure display is correct
	switch newTask.Computed.DisplayType {
	case DisplayNone:
	case DisplayXorg:
		_, err := backendState.displays.GetXorgDisplay(newTask.Computed.DisplayName, scheduler)
		if err != nil {
			return false // TODO return proper error here, since now we have multiple different errors
		}
	default:
		panic("Not implemented")
	}

	// Assign unique ID to the process. Note it isn't part of the hash above.
	newTask.Computed.Id = backendState.scheduler.currentId
	scheduler.currentId++

	scheduler.tasks = append(scheduler.tasks, newTask)
	go ExecuteTask(newTask, backendState)
	return true
}

func TryUnscheduleTask(taskToRemove *Task, backendState *BackendState) bool {
	backendState.scheduler.lock.Lock()
	defer backendState.scheduler.lock.Unlock()

	var newTasks []*Task
	var numRemoved int

	for _, currTask := range backendState.scheduler.tasks {
		if currTask.Computed.Hash == taskToRemove.Computed.Hash {
			numRemoved++
		} else {
			newTasks = append(newTasks, currTask)
		}
	}

	backendState.scheduler.tasks = newTasks
	return numRemoved > 0 // TODO make some warning if it was greater than 1
}

func (scheduler *Scheduler) KillProcessesByDisplay(displayType DisplayType, displayName string) {
	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	for _, currTask := range scheduler.tasks {
		if currTask.Computed.DisplayName == displayName && currTask.Computed.DisplayType == displayType {
			currTask.Channels.StopChannel <- fmt.Sprintf("killing processes on display %v", displayName)
		}
	}
}

func ExecuteTask(task *Task, backendState *BackendState) {
	// Initialize per-task logger
	perTaskLogger := CreateFileLogger(task.OutFilePath)
	err := perTaskLogger.run()
	if err != nil {
		backendState.messages.Add(BackendMessageError, "failed to create per-process logger")
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

		if hasFlag(LogDeactivation | LogBackend | LogTask) {
			perTaskLogger.channel <- diagnosticMessage(content, hasFlag(LogFlagTaskSeparator))
		}
		if hasFlag(LogDeactivation | LogBackend) {
			severity := BackendMessageInfo
			if hasFlag(LogFlagErr) {
				severity = BackendMessageError
			}
			backendState.messages.Add(severity, content)
		}
		if hasFlag(LogDeactivation) {
			isTaskDeactivated = true
			task.Deactivate(content)
		}
	}
	logF := func(flags LogFlag, format string, args ...any) {
		content := fmt.Sprintf(format, args...)
		log(flags, content)
	}

	// TODO write LogTask with general info about the task

	// Execute the main loop until the task becomes deactivated.
	subsequentFailures := 0
	for !isTaskDeactivated {
		// Initialize the process struct
		cmdContext, cmdCancel := context.WithCancel(context.Background())
		defer cmdCancel()
		cmd := exec.CommandContext(cmdContext, task.Cmdline[0], task.Cmdline[1:]...)
		cmd.Dir = task.Cwd
		cmd.Env = task.Env
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to create stdout pipe")
			break
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to create stderr pipe")
			break
		}

		// Start the process
		err = cmd.Start()
		if err != nil {
			log(LogDeactivation|LogFlagErr, "Failed to create stdout pipe")
			break
		}
		log(LogTask, "Process started")

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

		// Wait for the process in a separate goroutine and signal when it ends. It's important to first wait for the
		// goroutines streaming the output. Otherwise, cmd.Wait() will close the pipes leading to a race condition.
		processResultChannel := make(chan int)
		go func() {
			pipeWaitGroup.Wait()

			status := 0
			err := cmd.Wait()

			if err != nil {
				status = err.(*exec.ExitError).ExitCode()
			}
			processResultChannel <- status
		}()

		// Block until something happens
		select {
		case exitCode := <-processResultChannel:
			// Process ended.
			logF(LogTask|LogFlagTaskSeparator, "Process ended with code %v", exitCode)
			if exitCode == 0 {
				subsequentFailures = 0
			} else {
				subsequentFailures++
			}
		case <-perTaskLogger.errorChannel:
			// Logger failed. We don't want to execute processes without logging. Kill the process and return error.
			log(LogDeactivation|LogFlagErr, "Failed logging")
		case reason := <-task.Channels.StopChannel:
			logF(LogDeactivation, "Process killed (%v)", reason)
		}

		// Handle breaks from the main loop
		if task.MaxSubsequentFailures >= 0 && subsequentFailures >= task.MaxSubsequentFailures {
			logF(LogDeactivation, "Task reached subsequent failure count limit of %v", task.MaxSubsequentFailures)
			break
		}
	}
}
