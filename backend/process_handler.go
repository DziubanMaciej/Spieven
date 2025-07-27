package backend

import (
	"context"
	"fmt"
	"hash/fnv"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type DisplayType byte

const (
	DisplayNone DisplayType = iota
	DisplayXorg
	DisplayWayland
)

type ProcessDescription struct {
	Cmdline               []string
	Cwd                   string
	Env                   []string
	OutFilePath           string
	MaxSubsequentFailures int
	UserIndex             int

	Computed struct {
		Id          int
		Hash        int
		DisplayType DisplayType
		DisplayName string
	}

	Channels struct {
		StopChannel chan string
	}
}

func (desc *ProcessDescription) ComputeHash() {
	h := fnv.New32a()

	writeInt := func(val int) {
		h.Write([]byte(strconv.Itoa(val)))
	}
	writeString := func(val string) {
		h.Write([]byte(val))
	}
	writeStrings := func(val []string) {
		for _, s := range val {
			writeString(s)
		}
	}

	writeStrings(desc.Cmdline)
	writeString(desc.Cwd)
	writeString(desc.OutFilePath)
	writeInt(desc.MaxSubsequentFailures)
	writeInt(desc.UserIndex)

	desc.Computed.Hash = int(h.Sum32())
}

func (desc *ProcessDescription) ComputeDisplay() {
	// Search the env variable for display-related settings.
	var displayVar string
	var waylandDisplayVar string
	for _, currentVar := range desc.Env {
		parts := strings.SplitN(currentVar, "=", 2)
		switch parts[0] {
		case "DISPLAY":
			displayVar = parts[1]
		case "WAYLAND_DISPLAY":
			waylandDisplayVar = parts[1]
		}
	}

	// Select one of three DisplayType options based on those envs. Technically, if app has both DISPLAY and WAYLAND_DISPLAY
	// it could choose either one, e.g. based on argv or some config file. In general we cannot know which one it'll use.
	// It could even use both. Just prefer Wayland for simplicity.
	if waylandDisplayVar != "" {
		desc.Computed.DisplayType = DisplayWayland
		desc.Computed.DisplayName = waylandDisplayVar
	} else if displayVar != "" {
		desc.Computed.DisplayType = DisplayXorg
		desc.Computed.DisplayName = displayVar
	} else {
		desc.Computed.DisplayType = DisplayNone
		desc.Computed.DisplayName = ""
	}
}

func (desc *ProcessDescription) InitChannels() {
	desc.Channels.StopChannel = make(chan string, 1)
}

type RunningProcesses struct {
	processes []ProcessDescription
	currentId int
	lock      sync.Mutex
}

func TryRegisterProcess(newDesc *ProcessDescription, backendState *BackendState) bool {
	backendState.processes.lock.Lock()
	defer backendState.processes.lock.Unlock()

	// Calculate process hash and skip registering if we already have it.
	newDesc.ComputeHash()
	newDesc.ComputeDisplay()
	newDesc.InitChannels()
	for _, currDesc := range backendState.processes.processes {
		if currDesc.Computed.Hash == newDesc.Computed.Hash {
			return false
		}
	}

	// Ensure display is correct
	switch newDesc.Computed.DisplayType {
	case DisplayNone:
	case DisplayXorg:
		_, err := backendState.displays.GetXorgDisplay(newDesc.Computed.DisplayName, &backendState.processes)
		if err != nil {
			return false // TODO return proper error here, since now we have multiple different errors
		}
	default:
		panic("Not implemented")
	}

	// Assign unique ID to the process. Note it isn't part of the hash above.
	newDesc.Computed.Id = backendState.processes.currentId
	backendState.processes.currentId++

	backendState.processes.processes = append(backendState.processes.processes, *newDesc)
	go HandleProcess(newDesc, &backendState.processes, &backendState.messages)
	return true
}

func (processes *RunningProcesses) TryUnregisterProcess(newDesc *ProcessDescription) bool {
	processes.lock.Lock()
	defer processes.lock.Unlock()

	var newProcesses []ProcessDescription
	var processesRemoved int

	for _, currDesc := range processes.processes {
		if currDesc.Computed.Hash == newDesc.Computed.Hash {
			processesRemoved++
		} else {
			newProcesses = append(newProcesses, currDesc)
		}
	}

	processes.processes = newProcesses
	return processesRemoved > 0 // TODO make some warning if it was greater than 1
}

func (processes *RunningProcesses) KillProcessesByDisplay(displayType DisplayType, displayName string) {
	processes.lock.Lock()
	defer processes.lock.Unlock()

	for _, currDesc := range processes.processes {
		if currDesc.Computed.DisplayName == displayName && currDesc.Computed.DisplayType == displayType {
			currDesc.Channels.StopChannel <- fmt.Sprintf("killing processes on display %v", displayName)
		}
	}
}

func HandleProcess(processDescription *ProcessDescription, processes *RunningProcesses, backendMessages *BackendMessages) {
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
		cmd.Env = processDescription.Env
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
			backendMessages.Add(BackendMessageError, "Failed logging")
			break MainLoop
		case reason := <-processDescription.Channels.StopChannel:
			backendMessages.AddF(BackendMessageInfo, "Process killed (%v)", reason)
			break MainLoop
		}

		// Handle breaks from the main loop
		if processDescription.MaxSubsequentFailures >= 0 && subsequentFailures >= processDescription.MaxSubsequentFailures {
			log.channel <- diagnosticMessageF("Process reached subsequent failure count limit of %v. Exiting", processDescription.MaxSubsequentFailures)
			break
		}
	}

	if processes.TryUnregisterProcess(processDescription) {
		backendMessages.Add(BackendMessageInfo, "Unregistered process")
	}
}
