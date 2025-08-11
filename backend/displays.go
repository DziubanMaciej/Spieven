package backend

import (
	"fmt"
	"os"
	"os/exec"
	"supervisor/common"
	"supervisor/common/types"
	"syscall"

	"sync"
)

type Displays struct {
	xorgDisplays []*XorgDisplay
	lock         sync.Mutex

	_ common.NoCopy
}

func GetXorgDisplay(name string, backendState *BackendState) (*XorgDisplay, error) {
	displays := &backendState.displays

	displays.lock.Lock()
	defer displays.lock.Unlock()

	for _, currDisplay := range displays.xorgDisplays {
		if !currDisplay.IsDeactivated && currDisplay.Name == name {
			return currDisplay, nil
		}
	}

	newDisplay, err := NewXorgDisplay(name, backendState)
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

func NewXorgDisplay(name string, backendState *BackendState) (*XorgDisplay, error) {
	// First try to connect to XServer. If it cannot be done, the passed DISPLAY value is invalid
	dpy := common.TryConnectXorg(name)
	if dpy == nil {
		return nil, fmt.Errorf("cannot connect to xorg")
	}
	common.DisconnectXorg(dpy)

	// Run watchxorg. If this command ends, it will mean XServer has stopped working.
	spievenBinary := os.Args[0]
	cmd := exec.CommandContext(backendState.context, spievenBinary, "internal", "watchxorg", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // sets the child to a new process group, to avoid forwarding ctrl+C to it
	}
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start Spieven watchxorg")
	}

	result := XorgDisplay{
		Name: name,
	}

	backendState.StartGoroutine(func() {
		cmd.Wait()
		if backendState.IsContextKilled() {
			return
		}

		// Display is closed. Stop all tasks using it.
		backendState.displays.lock.Lock()
		backendState.scheduler.lock.Lock()
		result.IsDeactivated = true
		backendState.scheduler.StopTasksByDisplay(types.DisplaySelectionTypeXorg, name)
		backendState.scheduler.lock.Unlock()
		backendState.displays.lock.Unlock()

		// TODO implement a more sophisticated display termination: wait for some time before killing tasks to give them time to finish gracefully. Make it a backend's parameter.
	})

	return &result, nil
}
