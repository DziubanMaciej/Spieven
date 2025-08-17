package display

import (
	"errors"
	i "spieven/backend/interfaces"
	"spieven/common"
)

type Displays struct {
	xorgDisplays []*XorgDisplay
	lock         common.CheckedLock

	xorgSupported bool
	_             common.NoCopy
}

func CreateDisplays(messages i.IMessages) *Displays {
	xorgSupported := true
	xorgErr := common.LoadXorgLibs()
	if xorgErr != nil {
		xorgSupported = false
		messages.Add(i.BackendMessageInfo, nil, "Xorg libraries could not be loaded. Tasks with xorg display will not be accepted")
	}

	return &Displays{
		xorgSupported: xorgSupported,
	}
}

func (displays *Displays) Cleanup() {
	common.UnloadXorgLibs()
}

func (displays *Displays) InitXorgDisplay(name string, scheduler i.IScheduler, goroutines i.IGoroutines) error {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	if !displays.xorgSupported {
		return errors.New("xorg not supported")
	}

	for _, currDisplay := range displays.xorgDisplays {
		if !currDisplay.IsDeactivated && currDisplay.Name == name {
			return nil
		}
	}

	newDisplay, err := NewXorgDisplay(name, &displays.lock, scheduler, goroutines)
	if err == nil {
		displays.xorgDisplays = append(displays.xorgDisplays, newDisplay)
	}

	return err
}

func (displays *Displays) Trim() {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	var newDisplays []*XorgDisplay

	for _, currDisplay := range displays.xorgDisplays {
		if !currDisplay.IsDeactivated {
			newDisplays = append(newDisplays, currDisplay)
		}
	}

	displays.xorgDisplays = newDisplays
}
