package backend

import (
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"os"
	"spieven/common"
	"spieven/common/types"
	"strconv"
	"strings"
)

// Task struct describes a command that is scheduled to be running in background. For each Task Spieven creates a
// goroutine that constantly monitors its state, reruns it if neccessary and logs what's happening to the Task to
// a file.
//
// User of this struct should only fill fields outside of any sub-structs. Sub-structs contain data filled upon
// scheduling task or dynamically during execution.
type Task struct {
	Cmdline               []string
	Cwd                   string
	Env                   []string
	DelayAfterSuccessMs   int
	DelayAfterFailureMs   int
	MaxSubsequentFailures int
	FriendlyName          string
	CaptureStdout         bool
	Display               types.DisplaySelection
	Tags                  []string

	Computed struct {
		Id          int
		OutFilePath string
		LogLabel    string

		Hash            int
		NameDisplayHash int
	}

	Channels struct {
		StopChannel    chan string   `json:"-"`
		RefreshChannel chan struct{} `json:"-"`
	}

	Dynamic struct {
		RunCount               int
		FailureCount           int
		SubsequentFailureCount int
		LastExitValue          int
		LastStdoutFilePath     string
		IsDeactivated          bool
		DeactivatedReason      string
	}

	_ common.NoCopy
}

func (task *Task) Init(id int, outFilePath string) {
	// Set some derived values
	task.Computed.Id = id
	task.Computed.OutFilePath = outFilePath
	task.Computed.LogLabel = task.ComputeLogLabel(id)
	if task.Display.Type == types.DisplaySelectionTypeNone {
		task.Display = task.ComputeDisplayFromEnv()
	}
	common.SetDisplayEnvVarsForSubprocess(task.Display, &task.Env)

	// Create channels used for communicating with the task
	task.Channels.StopChannel = make(chan string, 1)
	task.Channels.RefreshChannel = make(chan struct{})

	// Reset some dynamic state in case we're reactivating a deactivated task
	task.Dynamic.SubsequentFailureCount = 0
	task.Dynamic.IsDeactivated = false
	task.Dynamic.DeactivatedReason = ""

	// Compute hashes for comparing tasks
	task.Computed.Hash, task.Computed.NameDisplayHash = task.ComputeHashes()

}

func (task *Task) ComputeHashes() (int, int) {
	var h hash.Hash32
	writeInt := func(val int) {
		h.Write([]byte(strconv.Itoa(val)))
	}
	writeBool := func(val bool) {
		if val {
			writeInt(1)
		} else {
			writeInt(0)
		}
	}
	writeString := func(val string) {
		h.Write([]byte(val))
	}
	writeStrings := func(val []string) {
		for _, s := range val {
			writeString(s)
		}
	}

	// This hash includes all parameters passed by the frontend and display information pulled from env
	h = fnv.New32a()
	writeStrings(task.Cmdline)
	writeString(task.Cwd)
	writeInt(task.MaxSubsequentFailures)
	writeString(task.FriendlyName)
	writeBool(task.CaptureStdout)
	writeStrings(task.Tags)
	writeInt(int(task.Display.Type))
	writeString(task.Display.Name)
	hash1 := int(h.Sum32())

	// This hash includes user-passed friendly name and display information pulled from env. It ensures
	// that we only have one task with a given name per display.
	h = fnv.New32a()
	writeString(task.FriendlyName)
	writeInt(int(task.Display.Type))
	writeString(task.Display.Name)
	hash2 := int(h.Sum32())

	return hash1, hash2
}

func (task *Task) ComputeDisplayFromEnv() types.DisplaySelection {
	// Search the env variable for display-related settings.
	var xorgDisplayVar string
	var waylandDisplayVar string
	for _, currentVar := range task.Env {
		parts := strings.SplitN(currentVar, "=", 2)
		switch parts[0] {
		case "DISPLAY":
			xorgDisplayVar = parts[1]
		case "WAYLAND_DISPLAY":
			waylandDisplayVar = parts[1]
		}
	}

	// Select one of three DisplayType options based on those envs. Technically, if app has both DISPLAY and WAYLAND_DISPLAY
	// it could choose either one, e.g. based on argv or some config file. In general we cannot know which one it'll use.
	// It could even use both. Just prefer Wayland for simplicity.
	if waylandDisplayVar != "" {
		return types.DisplaySelection{
			Type: types.DisplaySelectionTypeWayland,
			Name: waylandDisplayVar,
		}
	} else if xorgDisplayVar != "" {
		return types.DisplaySelection{
			Type: types.DisplaySelectionTypeXorg,
			Name: xorgDisplayVar,
		}
	} else {
		return types.DisplaySelection{
			Type: types.DisplaySelectionTypeHeadless,
			Name: "",
		}
	}
}

func (task *Task) ComputeLogLabel(id int) string {
	return fmt.Sprintf("task id=%v, %v", id, task.FriendlyName)
}

func (task *Task) ReadLastStdout() (stdout string, err error) {
	if !task.CaptureStdout {
		err = errors.New("stdout was not captured")
		return
	}

	if task.Dynamic.LastStdoutFilePath == "" {
		err = errors.New("no stdout saved")
		return
	}

	var stdoutFile *os.File
	stdoutFile, err = os.OpenFile(task.Dynamic.LastStdoutFilePath, os.O_RDONLY, 0644)
	if err != nil {
		err = errors.New("failed opening stdout file")
		return
	}

	stdout, err = common.ReadUntilEof(stdoutFile)
	if err != nil {
		err = errors.New("error reading stdout file")
		return
	}

	return
}
