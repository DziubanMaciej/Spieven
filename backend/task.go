package backend

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"supervisor/common"
	"sync"
	"time"
)

// Task struct describes a process that is scheduled to be running in background. For each Task Spieven creates a
// goroutine that constantly monitors its state, reruns it if neccessary and logs what's happening to the Task to
// a file.
//
// User of this struct should only fill fields outside of any sub-structs. Sub-structs contain data filled upon
// scheduling task or dynamically during execution.
type Task struct {
	Cmdline               []string
	Cwd                   string
	Env                   []string
	OutFilePath           string
	MaxSubsequentFailures int
	UserIndex             int
	FriendlyName          string

	Computed struct {
		Id          int
		Hash        int
		DisplayType DisplayType
		DisplayName string
		LogLabel    string
	}

	Channels struct {
		StopChannel chan string
	}

	Dynamic struct {
		Lock              sync.Mutex // TODO Remove this, lock the whole scheduler
		IsDeactivated     bool
		DeactivatedReason string
		DeactivatedTime   time.Time
	}

	_ common.NoCopy
}

type DisplayType byte

const (
	DisplayNone DisplayType = iota
	DisplayXorg
	DisplayWayland
)

func (task *Task) Init(id int) {
	task.Computed.Id = id
	task.Computed.Hash = task.ComputeHash()
	task.Computed.DisplayType, task.Computed.DisplayName = task.ComputeDisplay()
	task.Computed.LogLabel = task.ComputeLogLabel(id)

	task.Channels.StopChannel = task.CreateStopChannel()
}

func (task *Task) ComputeHash() int {
	h := fnv.New32a()

	writeInt := func(val int) {
		h.Write([]byte(strconv.Itoa(val)))
	}
	writeString := func(val string) {
		h.Write([]byte(val))
	}
	writeStrings := func(val []string) {
		for _, s := range val {
			writeString(s)
		}
	}

	writeStrings(task.Cmdline)
	writeString(task.Cwd)
	writeString(task.OutFilePath)
	writeInt(task.MaxSubsequentFailures)
	writeInt(task.UserIndex)

	return int(h.Sum32())
}

func (task *Task) ComputeDisplay() (DisplayType, string) {
	// Search the env variable for display-related settings.
	var displayVar string
	var waylandDisplayVar string
	for _, currentVar := range task.Env {
		parts := strings.SplitN(currentVar, "=", 2)
		switch parts[0] {
		case "DISPLAY":
			displayVar = parts[1]
		case "WAYLAND_DISPLAY":
			waylandDisplayVar = parts[1]
		}
	}

	// Select one of three DisplayType options based on those envs. Technically, if app has both DISPLAY and WAYLAND_DISPLAY
	// it could choose either one, e.g. based on argv or some config file. In general we cannot know which one it'll use.
	// It could even use both. Just prefer Wayland for simplicity.
	if waylandDisplayVar != "" {
		return DisplayWayland, waylandDisplayVar
	} else if displayVar != "" {
		return DisplayXorg, displayVar
	} else {
		return DisplayNone, ""
	}
}

func (task *Task) ComputeLogLabel(id int) string {
	label := task.FriendlyName
	if label == "" {
		label = task.Cmdline[0]
	}
	return fmt.Sprintf("task id=%v, %v", id, label)
}

func (task *Task) CreateStopChannel() chan string {
	return make(chan string, 1)
}

func (task *Task) Deactivate(reason string) {
	task.Dynamic.Lock.Lock()
	task.Dynamic.IsDeactivated = true
	task.Dynamic.DeactivatedReason = reason
	task.Dynamic.DeactivatedTime = time.Now()
	task.Dynamic.Lock.Unlock()
}
