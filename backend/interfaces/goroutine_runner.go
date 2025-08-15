package interfaces

import "context"

type GoroutineRunner interface {
	GetContext() *context.Context
	IsContextKilled() bool
	StartGoroutine(body func())
	StartGoroutineAfterContextKill(body func())
}
