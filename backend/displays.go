package backend

import (
	"fmt"
	"os"
	"os/exec"
	"supervisor/watchxorg"
	"sync"
)

type Displays struct {
	xorgDisplays []XorgDisplay
	lock         sync.Mutex
}

func (displays *Displays) GetXorgDisplay(name string, processes *RunningProcesses) (*XorgDisplay, error) {
	displays.lock.Lock()
	defer displays.lock.Unlock()

	for _, currDisplay := range displays.xorgDisplays {
		if currDisplay.name == name {
			return &currDisplay, nil
		}
	}

	newDisplay, err := NewXorgDisplay(name, processes)
	if err == nil {
		displays.xorgDisplays = append(displays.xorgDisplays, *newDisplay)
	}

	return newDisplay, err
}

type XorgDisplay struct {
	name string
}

func NewXorgDisplay(name string, processes *RunningProcesses) (*XorgDisplay, error) {
	// First try to connect to XServer. If it cannot be done, the passed DISPLAY value is invalid
	dpy := watchxorg.TryConnectXorg(name)
	if dpy == nil {
		return nil, fmt.Errorf("cannot connect to xorg")
	}
	watchxorg.DisconnectXorg(dpy)

	// Run watchxorg. If this command ends, it will mean XServer has stopped working.
	spievenBinary := os.Args[0]
	cmd := exec.Command(spievenBinary, "watchxorg", name)
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start Spieven watchxorg")
	}

	go func() {
		cmd.Wait()

		// Display is closed. Notify all the process handlers
		processes.KillProcessesByDisplay(DisplayXorg, name)
	}()

	result := XorgDisplay{
		name: name,
	}
	return &result, nil
}
