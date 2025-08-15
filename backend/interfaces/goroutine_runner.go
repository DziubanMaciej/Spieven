package interfaces

import "context"

type IGoroutines interface {
	GetContext() *context.Context
	IsContextKilled() bool
	StartGoroutine(body func())
	StartGoroutineAfterContextKill(body func())
}
