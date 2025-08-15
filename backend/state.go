package backend

import (
	"spieven/common"
	"spieven/common/buildopts"
	"time"
)

// BackendState stores global state shared by whole backend, i.e by all frontend connections and running tasks
// handlers. It consists of structs containing synchronized methods, to allow access from different goroutines.
type BackendState struct {
	sync      *BackendSync
	files     *FilePathProvider
	messages  *BackendMessages
	scheduler Scheduler
	displays  Displays

	handshakeValue uint64

	_ common.NoCopy
}

func CreateBackendState(frequentTrim bool) (*BackendState, error) {
	sync, err := CreateBackendSync()
	if err != nil {
		return nil, err
	}

	files, err := CreateFilePathProvider(buildopts.DefaultPort)
	if err != nil {
		return nil, err
	}

	messages, err := CreateBackendMessages(files.GetBackendMessagesLogFile())
	if err != nil {
		return nil, err
	}

	backendState := BackendState{
		sync:     sync,
		files:    files,
		messages: messages,
	}
	backendState.StartTrimGoroutine(frequentTrim)
	backendState.StartCleanupGorotuine()

	return &backendState, nil
}

func (state *BackendState) StartTrimGoroutine(frequentTrim bool) {
	const maxMessageAge = time.Hour * 2
	const maxTaskAge = time.Minute * 5
	const maxTrimInterval = min(maxMessageAge, maxTaskAge) / 2

	trimInterval := maxTrimInterval
	if frequentTrim {
		trimInterval = time.Millisecond * 500
	}

	body := func() {
		for {
			select {
			case <-state.sync.context.Done():
				return
			case <-time.After(trimInterval):
				state.messages.Trim(maxMessageAge)

				state.scheduler.lock.Lock()
				state.scheduler.Trim(state.messages, state.files)
				state.scheduler.lock.Unlock()

				state.displays.Trim()
			}
		}
	}
	state.sync.StartGoroutine(body)
}

func (state *BackendState) StartCleanupGorotuine() {
	body := func() {
		state.files.Cleanup()
		state.messages.Cleanup()
	}
	state.sync.StartGoroutineAfterContextKill(body)
}
