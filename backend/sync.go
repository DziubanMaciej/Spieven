package backend

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type BackendSync struct {
	context     context.Context
	killContext context.CancelFunc
	waitGroup   sync.WaitGroup
}

func CreateBackendSync() (*BackendSync, error) {
	context, cancelContext := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	sync := BackendSync{
		context:     context,
		killContext: cancelContext,
	}
	return &sync, nil
}

func (sync *BackendSync) IsContextKilled() bool {
	return sync.context.Err() != nil
}

func (sync *BackendSync) GetContext() *context.Context {
	return &sync.context
}

func (sync *BackendSync) StartGoroutine(body func()) {
	sync.waitGroup.Add(1)
	go func() {
		body()
		sync.waitGroup.Done()
	}()
}

func (sync *BackendSync) StartGoroutineAfterContextKill(body func()) {
	sync.waitGroup.Add(1)
	go func() {
		<-sync.context.Done()
		body()
		sync.waitGroup.Done()
	}()
}
