package interfaces

import "spieven/common/types"

type IScheduler interface {
	Lock()
	Unlock()
	StopTasksByDisplay(displayType types.DisplaySelectionType, displayName string)
}
