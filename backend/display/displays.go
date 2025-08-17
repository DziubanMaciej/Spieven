package display

import (
	"errors"
	i "spieven/backend/interfaces"
	"spieven/common"
	"spieven/common/types"
	"time"
)

type Displays struct {
	killGracePeriod  time.Duration
	xorgSupported    bool
	waylandSupported bool
	displays         []*Display

	lock common.CheckedLock
	_    common.NoCopy
}

func CreateDisplays(messages i.IMessages, killGracePeriod time.Duration) *Displays {
	xorgSupported := true
	xorgErr := common.LoadXorgLibs()
	if xorgErr != nil {
		xorgSupported = false
		messages.Add(i.BackendMessageInfo, nil, "Xorg libraries could not be loaded. Tasks with xorg display will not be accepted")
	}

	waylandSupported := true
	waylandErr := common.LoadWaylandLibs()
	if waylandErr != nil {
		waylandSupported = false
		messages.Add(i.BackendMessageInfo, nil, "Wayland libraries could not be loaded. Tasks with wayland display will not be accepted")
	}

	return &Displays{
		killGracePeriod:  killGracePeriod,
		xorgSupported:    xorgSupported,
		waylandSupported: waylandSupported,
	}
}

func (displays *Displays) Cleanup() {
	common.UnloadXorgLibs()
	common.UnloadWaylandLibs()
}

func (displays *Displays) InitDisplay(displaySelection types.DisplaySelection, scheduler i.IScheduler, goroutines i.IGoroutines, messages i.IMessages) error {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	// Validate support for passed display type
	switch displaySelection.Type {
	case types.DisplaySelectionTypeXorg:
		if !displays.xorgSupported {
			return errors.New("xorg not supported")
		}
	case types.DisplaySelectionTypeWayland:
		if !displays.waylandSupported {
			return errors.New("wayland not supported")
		}
	default:
		return errors.New("invalid display type")
	}

	// Search whether we have existing display matching passed args
	for _, currDisplay := range displays.displays {
		if !currDisplay.isDeactivated && currDisplay.selection == displaySelection {
			return nil
		}
	}

	// Create a new display and store it
	newDisplay, err := newDisplay(displaySelection, &displays.lock, scheduler, goroutines, messages, displays.killGracePeriod)
	if err != nil {
		return err
	}
	displays.displays = append(displays.displays, newDisplay)

	return nil
}

func (displays *Displays) Trim() {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	var newDisplays []*Display

	for _, currDisplay := range displays.displays {
		if !currDisplay.isDeactivated {
			newDisplays = append(newDisplays, currDisplay)
		}
	}

	displays.displays = newDisplays
}
