package interfaces

import "spieven/common/types"

type IScheduler interface {
	Lock()
	Unlock()
	StopTasksByDisplay(displaySelection types.DisplaySelection)
}
