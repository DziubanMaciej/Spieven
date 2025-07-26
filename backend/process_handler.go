package backend

import (
	"context"
	"os/exec"
	"sync"
)

type ProcessDescription struct {
	Cmdline               []string
	Cwd                   string
	OutFilePath           string
	MaxSubsequentFailures int
	Hash                  int
}

func (desc *ProcessDescription) CalculateHash() {
	desc.Hash = 123 // TODO
}

type RunningProcesses struct {
	processes []ProcessDescription
	lock      sync.Mutex
}

func (processes *RunningProcesses) TryRegisterProcess(newDesc *ProcessDescription, backendMessages *BackendMessages) bool {
	processes.lock.Lock()
	defer processes.lock.Unlock()

	// Search whether we already have a process like this
	for _, currDesc := range processes.processes {
		if currDesc.Hash == newDesc.Hash {
			return false
		}
	}

	processes.processes = append(processes.processes, *newDesc)
	go HandleProcess(*newDesc, processes, backendMessages)
	return true
}

func (processes *RunningProcesses) TryUnregisterProcess(newDesc *ProcessDescription) bool {
	processes.lock.Lock()
	defer processes.lock.Unlock()

	var newProcesses []ProcessDescription
	var processesRemoved int

	for _, currDesc := range processes.processes {
		if currDesc.Hash == newDesc.Hash {
			processesRemoved++
		} else {
			newProcesses = append(newProcesses, currDesc)
		}
	}

	processes.processes = newProcesses
	return processesRemoved > 0 // TODO make some warning if it was greater than 1
}

func HandleProcess(processDescription ProcessDescription, processes *RunningProcesses, backendMessages *BackendMessages) {
	// Initialize per-process logger
	log := CreateFileLogger(processDescription.OutFilePath)
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
		cmd := exec.CommandContext(cmdContext, processDescription.Cmdline[0], processDescription.Cmdline[1:]...)
		cmd.Dir = processDescription.Cwd
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
		log.channel <- diagnosticMessageF("Process \"%v\" started", processDescription.Cmdline[0])

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
			cmdCancel()
			backendMessages.Add(BackendMessageError, "failed logging")
			break MainLoop
		}

		// Handle breaks from the main loop
		if processDescription.MaxSubsequentFailures >= 0 && subsequentFailures >= processDescription.MaxSubsequentFailures {
			log.channel <- diagnosticMessageF("Process reached subsequent failure count limit of %v. Exiting", processDescription.MaxSubsequentFailures)
			break
		}
	}

	if processes.TryUnregisterProcess(&processDescription) {
		backendMessages.Add(BackendMessageInfo, "Unregistered process")
	}
}
