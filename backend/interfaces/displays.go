package interfaces

import "spieven/common/types"

type IDisplays interface {
	InitDisplay(displaySelection types.DisplaySelection, scheduler IScheduler, goroutines IGoroutines) error
}
