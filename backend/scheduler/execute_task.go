package scheduler

import (
	"context"
	"fmt"
	"os/exec"
	i "spieven/backend/interfaces"
	"spieven/common"
	"sync"
	"time"
)

func ExecuteTask(
	task *Task,
	schedulerLock *common.CheckedLock,
	files i.IFiles,
	goroutines i.IGoroutines,
	messages i.IMessages,
) {
	// Initialize per-task logger
	perTaskLogger := CreateFileLogger(files, goroutines, task.Computed.Id, task.CaptureStdout)
	err := perTaskLogger.run()
	if err != nil {
		messages.Add(i.BackendMessageError, task, "failed to create per-task logger")
		return
	}
	defer perTaskLogger.stop()

	// Copy the dynamic portion of task structure. Updates to it must be synchronized. We will be updating a local
	// copy and assign it to the actual task struct under a lock in one go every time something changes. Technically
	// this initial copy doesn't need a lock, because no other routine than ExecuteTask should ever change task.Dynamic.
	// But, for completeness we're still locking.
	schedulerLock.Lock()
	shadowDynamicState := task.Dynamic
	schedulerLock.Unlock()

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
			severity := i.BackendMessageInfo
			if hasFlag(LogFlagErr) {
				severity = i.BackendMessageError
			}
			messages.Add(severity, task, content)
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
		cmdContext, cmdCancel := context.WithCancel(*goroutines.GetContext())
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
		goroutines.StartGoroutine(func() {
			perTaskLogger.streamOutput(stdoutPipe)
			pipeWaitGroup.Done()
		})
		goroutines.StartGoroutine(func() {
			perTaskLogger.streamOutput(stderrPipe)
			pipeWaitGroup.Done()
		})

		// Wait for the command in a separate goroutine and signal when it ends. It's important to first wait for the
		// goroutines streaming the output. Otherwise, cmd.Wait() will close the pipes leading to a race condition.
		commandResultChannel := make(chan int, 1)
		goroutines.StartGoroutine(func() {
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
			shadowDynamicState.LastExitValue = exitCode // TODO this is wrong for test_script.sh, 127 is returned
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
			shadowDynamicState.SubsequentFailureCount = 0
		} else {
			shadowDynamicState.FailureCount++
			shadowDynamicState.SubsequentFailureCount++
		}

		// Handle MaxSubsequentFailures
		if task.MaxSubsequentFailures >= 0 && shadowDynamicState.SubsequentFailureCount >= task.MaxSubsequentFailures {
			logF(LogDeactivation, "Task reached subsequent failure count limit of %v.", task.MaxSubsequentFailures)
		}

		// Update dynamic state
		schedulerLock.Lock()
		task.Dynamic = shadowDynamicState
		schedulerLock.Unlock()

		// Perform delay between command executions
		if !shadowDynamicState.IsDeactivated {
			delay := task.DelayAfterFailureMs
			if commandSuccess {
				delay = task.DelayAfterSuccessMs
			}

			// Start a timer
			timer := time.NewTimer(time.Millisecond * time.Duration(delay))
			defer timer.Stop()

			// Wait for either the timer or the stop channel
			select {
			case <-timer.C:
			case <-task.Channels.RefreshChannel:
			case reason := <-task.Channels.StopChannel:
				logF(LogDeactivation, "Task killed (%v).", reason)
			case <-cmdContext.Done():
				logF(LogTask|LogDeactivation, "Backend killed.")
			}
		}
	}

	// Update dynamic state in case we broke from the loop
	schedulerLock.Lock()
	task.Dynamic = shadowDynamicState
	schedulerLock.Unlock()
}
