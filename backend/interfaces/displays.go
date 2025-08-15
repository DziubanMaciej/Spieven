package interfaces

type IDisplays interface {
	InitXorgDisplay(name string, scheduler IScheduler, goroutines IGoroutines) error
}
