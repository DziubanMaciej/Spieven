package display

import (
	i "spieven/backend/interfaces"
	"spieven/common"
)

type Displays struct {
	xorgDisplays []*XorgDisplay
	lock         common.CheckedLock

	_ common.NoCopy
}

func (displays *Displays) InitXorgDisplay(name string, scheduler i.IScheduler, goroutines i.IGoroutines) error {
	displays.lock.Lock()
	defer displays.lock.Unlock()

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
