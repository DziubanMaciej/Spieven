package backend

import (
	"context"
	"os/exec"
)

type ProcessDescription struct {
	Cmdline               []string
	Cwd                   string
	OutFilePath           string
	MaxSubsequentFailures int
}

func HandleProcess(processDescription ProcessDescription) error {
	// Initialize logger
	log := CreateFileLogger(processDescription.OutFilePath)
	err := log.run()
	if err != nil {
		return err
	}
	defer log.stop()

	subsequentFailures := 0
	for {
		// Initialize the process struct
		cmdContext, cmdCancel := context.WithCancel(context.Background())
		defer cmdCancel()
		cmd := exec.CommandContext(cmdContext, processDescription.Cmdline[0], processDescription.Cmdline[1:]...)
		cmd.Dir = processDescription.Cwd
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		// Start the process
		err = cmd.Start()
		if err != nil {
			return err
		}
		log.channel <- diagnosticMessageF("Process \"%v\" started", processDescription.Cmdline[0])
		go log.streamOutput(stdoutPipe)
		go log.streamOutput(stderrPipe)

		// Wait for the process in the background
		processResultChannel := make(chan int)
		go func() {
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
		case loggingError := <-log.errorChannel:
			// Logger failed. We don't want to execute processes without logging. Kill the process and return error.
			cmdCancel()
			return loggingError
		}

		// Handle breaks from the main loop
		if processDescription.MaxSubsequentFailures >= 0 && subsequentFailures >= processDescription.MaxSubsequentFailures {
			log.channel <- diagnosticMessageF("Process reached subsequent failure count limit of %v. Exiting", processDescription.MaxSubsequentFailures)
			return nil // TODO error?
		}
	}
}
