package backend

import (
	"supervisor/common"
	"time"
)

// BackendState stores global state shared by whole backend, i.e by all frontend connections and running tasks
// handlers. It consists of structs containing synchronized methods, to allow access from different goroutines.
// TODO implement Cleanup() function. Handle Ctrl+C. Make sure all goroutines are stopped. Call files.Cleanup() to remove cached files
type BackendState struct {
	messages  *BackendMessages
	scheduler Scheduler
	displays  Displays
	files     *FilePathProvider

	handshakeValue uint64

	trimGoroutineStopChannel *chan struct{}

	_ common.NoCopy
}

func CreateBackendState() (*BackendState, error) {
	files, err := CreateFilePathProvider()
	if err != nil {
		return nil, err
	}

	messages, err := CreateBackendMessages(files.GetBackendMessagesLogFile())
	if err != nil {
		return nil, err
	}

	backendState := BackendState{
		files:    files,
		messages: messages,
	}
	backendState.StartTrimGoroutine()

	return &backendState, nil
}

func (state *BackendState) StartTrimGoroutine() {
	const maxMessageAge = time.Hour * 2
	const maxTaskAge = time.Minute * 5
	const trimInterval = min(maxMessageAge, maxTaskAge) / 2

	stopChannel := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-stopChannel:
				return
			case <-time.After(trimInterval):
				state.messages.Trim(maxMessageAge)
				state.scheduler.Trim(maxTaskAge, state.messages)
				state.displays.Trim()
			}
		}
	}()

	state.trimGoroutineStopChannel = &stopChannel
}

func (state *BackendState) StopTrimGoroutine() {
	if state.trimGoroutineStopChannel != nil {
		*state.trimGoroutineStopChannel <- struct{}{}
	}
}
