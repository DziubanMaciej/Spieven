package backend

import (
	"time"
)

// BackendState stores global state shared by whole backend, i.e by all frontend connections and running process
// handlers. It consists of structs containing synchronized methods, to allow access from different goroutines.
type BackendState struct {
	messages  BackendMessages
	scheduler Scheduler
	displays  Displays

	handshakeValue uint64
}

func (state *BackendState) StartTrimGoroutine() chan struct{} {
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
				state.scheduler.Trim(maxTaskAge, &state.messages)
			}
		}
	}()

	return stopChannel
}
