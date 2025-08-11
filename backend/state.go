package backend

import (
	"context"
	"os"
	"os/signal"
	"supervisor/common"
	"sync"
	"syscall"
	"time"
)

// BackendState stores global state shared by whole backend, i.e by all frontend connections and running tasks
// handlers. It consists of structs containing synchronized methods, to allow access from different goroutines.
type BackendState struct {
	messages  *BackendMessages
	scheduler Scheduler
	displays  Displays
	files     *FilePathProvider

	handshakeValue uint64

	context     context.Context
	killContext context.CancelFunc
	waitGroup   sync.WaitGroup

	_ common.NoCopy
}

func CreateBackendState(frequentTrim bool) (*BackendState, error) {
	files, err := CreateFilePathProvider()
	if err != nil {
		return nil, err
	}

	messages, err := CreateBackendMessages(files.GetBackendMessagesLogFile())
	if err != nil {
		return nil, err
	}

	context, cancelContext := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	backendState := BackendState{
		files:       files,
		messages:    messages,
		context:     context,
		killContext: cancelContext,
	}
	backendState.StartTrimGoroutine(frequentTrim)

	return &backendState, nil
}

func (state *BackendState) IsContextKilled() bool {
	return state.context.Err() != nil
}

func (state *BackendState) StartGoroutine(body func()) {
	state.waitGroup.Add(1)
	go func() {
		body()
		state.waitGroup.Done()
	}()
}

func (state *BackendState) StartGoroutineAfterContextKill(body func()) {
	state.waitGroup.Add(1)
	go func() {
		<-state.context.Done()
		body()
		state.waitGroup.Done()
	}()
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
			case <-state.context.Done():
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
	state.StartGoroutine(body)
}
