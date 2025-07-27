package backend

import (
	"fmt"
	"os"
	"os/exec"
	"supervisor/common"

	"sync"
)

type Displays struct {
	xorgDisplays []*XorgDisplay
	lock         sync.Mutex

	_ common.NoCopy
}

func (displays *Displays) GetXorgDisplay(name string, scheduler *Scheduler) (*XorgDisplay, error) {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	for _, currDisplay := range displays.xorgDisplays {
		if !currDisplay.IsDeactivated && currDisplay.Name == name {
			return currDisplay, nil
		}
	}

	newDisplay, err := NewXorgDisplay(name, displays, scheduler)
	if err == nil {
		displays.xorgDisplays = append(displays.xorgDisplays, newDisplay)
	}

	return newDisplay, err
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

type XorgDisplay struct {
	Name          string
	IsDeactivated bool

	_ common.NoCopy
}

func NewXorgDisplay(name string, displays *Displays, scheduler *Scheduler) (*XorgDisplay, error) {
	// First try to connect to XServer. If it cannot be done, the passed DISPLAY value is invalid
	dpy := common.TryConnectXorg(name)
	if dpy == nil {
		return nil, fmt.Errorf("cannot connect to xorg")
	}
	common.DisconnectXorg(dpy)

	// Run watchxorg. If this command ends, it will mean XServer has stopped working.
	spievenBinary := os.Args[0]
	cmd := exec.Command(spievenBinary, "watchxorg", name)
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start Spieven watchxorg")
	}

	result := XorgDisplay{
		Name: name,
	}

	go func() {
		cmd.Wait()

		// Display is closed. Notify all the process handlers
		displays.lock.Lock()
		result.IsDeactivated = true
		scheduler.KillProcessesByDisplay(DisplayXorg, name)
		displays.lock.Unlock()
	}()

	return &result, nil
}
