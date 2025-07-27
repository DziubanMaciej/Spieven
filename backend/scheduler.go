package backend

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type Scheduler struct {
	tasks     []Task
	currentId int
	lock      sync.Mutex
}

func TryScheduleTask(newTask *Task, backendState *BackendState) bool {
	scheduler := &backendState.scheduler

	scheduler.lock.Lock()
	defer scheduler.lock.Unlock()

	// Calculate internal properties including the task's hash. Skip scheduling if we already have it.
	newTask.Init()
	for _, currDesc := range scheduler.tasks {
		if currDesc.Computed.Hash == newTask.Computed.Hash {
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

	scheduler.tasks = append(scheduler.tasks, *newTask)
	go ExecuteTask(newTask, backendState)
	return true
}

func TryUnscheduleTask(taskToRemove *Task, backendState *BackendState) bool {
	backendState.scheduler.lock.Lock()
	defer backendState.scheduler.lock.Unlock()

	var newTasks []Task
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

	for _, currDesc := range scheduler.tasks {
		if currDesc.Computed.DisplayName == displayName && currDesc.Computed.DisplayType == displayType {
			currDesc.Channels.StopChannel <- fmt.Sprintf("killing processes on display %v", displayName)
		}
	}
}

func ExecuteTask(task *Task, backendState *BackendState) {
	backendMessages := &backendState.messages

	// Initialize per-process logger
	log := CreateFileLogger(task.OutFilePath)
	err := log.run()
	if err != nil {
		backendMessages.Add(BackendMessageError, "failed to create per-process logger")
		return
	}
	defer log.stop()

	subsequentFailures := 0
MainLoop:
	for {
		// Initialize the process struct
		cmdContext, cmdCancel := context.WithCancel(context.Background())
		defer cmdCancel()
		cmd := exec.CommandContext(cmdContext, task.Cmdline[0], task.Cmdline[1:]...)
		cmd.Dir = task.Cwd
		cmd.Env = task.Env
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			backendMessages.Add(BackendMessageError, "failed to create stdout pipe")
			break
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			backendMessages.Add(BackendMessageError, "failed to create stdout pipe")
			break
		}

		// Start the process
		err = cmd.Start()
		if err != nil {
			backendMessages.Add(BackendMessageError, "failed to start process")
			break
		}
		log.channel <- diagnosticMessageF("Process \"%v\" started", task.Cmdline[0])

		// Run pipe reading goroutines
		var pipeWaitGroup sync.WaitGroup
		pipeWaitGroup.Add(2)
		go func() {
			log.streamOutput(stdoutPipe)
			pipeWaitGroup.Done()
		}()
		go func() {
			log.streamOutput(stderrPipe)
			pipeWaitGroup.Done()
		}()

		// Wait for the process in the background
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
			log.channel <- diagnosticMessageF("Process ended with code %v", exitCode)
			log.channel <- emptyLinesMessage(3)
			if exitCode == 0 {
				subsequentFailures = 0
			} else {
				subsequentFailures++
			}
		case <-log.errorChannel:
			// Logger failed. We don't want to execute processes without logging. Kill the process and return error.
			backendMessages.Add(BackendMessageError, "Failed logging")
			break MainLoop
		case reason := <-task.Channels.StopChannel:
			backendMessages.AddF(BackendMessageInfo, "Process killed (%v)", reason)
			break MainLoop
		}

		// Handle breaks from the main loop
		if task.MaxSubsequentFailures >= 0 && subsequentFailures >= task.MaxSubsequentFailures {
			log.channel <- diagnosticMessageF("Process reached subsequent failure count limit of %v. Exiting", task.MaxSubsequentFailures)
			break
		}
	}

	if TryUnscheduleTask(task, backendState) {
		backendMessages.Add(BackendMessageInfo, "Unregistered process")
	}
}
