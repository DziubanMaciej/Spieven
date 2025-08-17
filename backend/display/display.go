package display

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	i "spieven/backend/interfaces"
	"spieven/common"
	"spieven/common/types"
	"syscall"
)

type Display struct {
	selection     types.DisplaySelection
	isDeactivated bool

	_ common.NoCopy
}

func newDisplay(
	displaySelection types.DisplaySelection,
	displaysLock *common.CheckedLock,
	scheduler i.IScheduler,
	goroutines i.IGoroutines,
) (*Display, error) {
	// First try to connect to the display server. If it cannot be done, the passed display name is invalid.
	var watcherProcessArg string
	switch displaySelection.Type {
	case types.DisplaySelectionTypeXorg:
		dpy, err := common.TryConnectXorg(displaySelection.Name)
		if err != nil {
			return nil, err
		}
		common.DisconnectXorg(dpy)
		watcherProcessArg = "watchxorg"
	case types.DisplaySelectionTypeWayland:
		dpy, err := common.TryConnectWayland(displaySelection.Name)
		if err != nil {
			return nil, err
		}
		common.DisconnecWayland(dpy)
		watcherProcessArg = "watchwayland"
	default:
		return nil, errors.New("invalid display type")
	}

	// Run watcher process. If this command ends, it will mean the display server has stopped working.
	spievenBinary := os.Args[0]
	cmd := exec.CommandContext(*goroutines.GetContext(), spievenBinary, "internal", watcherProcessArg, displaySelection.Name)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // sets the child to a new process group, to avoid forwarding ctrl+C to it
	}
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start \"spieven internal %v %v\"", watcherProcessArg, displaySelection.Name)
	}

	result := Display{
		selection:     displaySelection,
		isDeactivated: false,
	}

	goroutines.StartGoroutine(func() {
		cmd.Wait()
		if goroutines.IsContextKilled() {
			return
		}

		// Display is closed. Stop all tasks using it.
		displaysLock.Lock()
		scheduler.Lock()
		result.isDeactivated = true
		scheduler.StopTasksByDisplay(displaySelection)
		scheduler.Unlock()
		displaysLock.Unlock()

		// TODO implement a more sophisticated display termination: wait for some time before killing tasks to give them time to finish gracefully. Make it a backend's parameter.
	})

	return &result, nil
}
