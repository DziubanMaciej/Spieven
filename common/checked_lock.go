package common

import (
	"sync"
	"sync/atomic"
)

// Checked lock is a helper struct wrapping a mutex and a boolean variable specifying whether the mutex is acquired.
// It is not thread safe and isLocked field shouldn't be used for any control flow. The purpose of this struct is to
// verify that the lock is already taken by the current thread.
type CheckedLock struct {
	lock     sync.Mutex
	isLocked atomic.Bool
}

func (entity *CheckedLock) Lock() {
	entity.lock.Lock()
	entity.isLocked.Store(true)
}

func (entity *CheckedLock) Unlock() {
	entity.isLocked.Store(false)
	entity.lock.Unlock()
}

func (entity *CheckedLock) AssertLocked() {
	if !entity.isLocked.Load() {
		panic("Invalid locking detected")
	}
}
