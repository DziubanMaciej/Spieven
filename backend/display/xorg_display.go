package display

import (
	"fmt"
	"os"
	"os/exec"
	i "spieven/backend/interfaces"
	"spieven/common"
	"spieven/common/types"
	"syscall"
)

type XorgDisplay struct {
	Name          string
	IsDeactivated bool

	_ common.NoCopy
}

func NewXorgDisplay(
	name string,
	displaysLock *common.CheckedLock,
	scheduler i.IScheduler,
	goroutines i.IGoroutines,
) (*XorgDisplay, error) {
	// First try to connect to XServer. If it cannot be done, the passed DISPLAY value is invalid
	dpy, err := common.TryConnectXorg(name)
	if err != nil {
		return nil, err
	}
	common.DisconnectXorg(dpy)

	// Run watchxorg. If this command ends, it will mean XServer has stopped working.
	spievenBinary := os.Args[0]
	cmd := exec.CommandContext(*goroutines.GetContext(), spievenBinary, "internal", "watchxorg", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // sets the child to a new process group, to avoid forwarding ctrl+C to it
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start Spieven watchxorg")
	}

	result := XorgDisplay{
		Name: name,
	}

	goroutines.StartGoroutine(func() {
		cmd.Wait()
		if goroutines.IsContextKilled() {
			return
		}

		// Display is closed. Stop all tasks using it.
		displaysLock.Lock()
		scheduler.Lock()
		result.IsDeactivated = true
		scheduler.StopTasksByDisplay(types.DisplaySelectionTypeXorg, name)
		scheduler.Unlock()
		displaysLock.Unlock()

		// TODO implement a more sophisticated display termination: wait for some time before killing tasks to give them time to finish gracefully. Make it a backend's parameter.
	})

	return &result, nil
}
