package scheduler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	i "spieven/backend/interfaces"
	"spieven/common"
	"spieven/common/types"
)

type Scheduler struct {
	tasks     []*Task
	currentId int
	lock      common.CheckedLock

	_ common.NoCopy
}

func (scheduler *Scheduler) Lock()                 { scheduler.lock.Lock() }
func (scheduler *Scheduler) Unlock()               { scheduler.lock.Unlock() }
func (scheduler *Scheduler) GetTasks() []*Task     { return scheduler.tasks }
func (scheduler *Scheduler) IsValidId(id int) bool { return id < scheduler.currentId }

func (scheduler *Scheduler) Trim(messages i.IMessages, files i.IFiles) {
	scheduler.lock.AssertLocked()

	var tasksToKeep []*Task
	var tasksToDeactivate []*Task

	// Divide tasks we have in memory into still active and deactivated tasks
	for _, currTask := range scheduler.tasks {
		if currTask.Dynamic.IsDeactivated {
			tasksToDeactivate = append(tasksToDeactivate, currTask)
		} else {
			tasksToKeep = append(tasksToKeep, currTask)
		}
	}

	// Push deactivated tasks out to a file
	if len(tasksToDeactivate) > 0 {
		filePath := files.GetDeactivatedTasksFile()
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			messages.AddF(i.BackendMessageError, nil, "Failed to open %s. Cannot push deactivated tasks out of memory to a file.", filePath)
			tasksToKeep = scheduler.tasks // Keep all tasks, so we don't lose data
		} else {
			for _, currTask := range tasksToDeactivate {
				// We're saving this as ndjson - json objects delimeted by newlines. For obvious reasons no fields of
				// Task can contain newlines. User inputs are sanitized for newlines, so they shouldn't appear in the
				// task struct.
				serializedTask, err := json.Marshal(currTask)
				serializedTask = append(serializedTask, '\n')

				if err == nil {
					err = common.WriteBytesToWriter(file, serializedTask)
				}

				if err == nil {
					messages.Add(i.BackendMessageInfo, currTask, "Trimmed task")
				} else {
					messages.AddF(i.BackendMessageError, currTask, "Failed to trim task: %s", err)
					tasksToKeep = append(tasksToKeep, currTask)
				}
			}
			file.Close()
		}

	}

	// Keep still active tasks in memory
	scheduler.tasks = tasksToKeep
}

func (scheduler *Scheduler) ReadTrimmedTasks(messages i.IMessages, files i.IFiles) []*Task {
	scheduler.lock.AssertLocked()

	var result []*Task

	filePath := files.GetDeactivatedTasksFile()
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		messages.AddF(i.BackendMessageError, nil, "Failed reading trimmed tasks: %s", err.Error())
		return result
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var task Task
		err := json.Unmarshal(scanner.Bytes(), &task)
		if err != nil {
			messages.AddF(i.BackendMessageError, nil, "Failed decoding a task from %s: %s", filePath, err.Error())
			continue
		}

		result = append(result, &task)
	}

	return result
}

func (scheduler *Scheduler) ExtractDeactivatedTask(
	taskId int,
	files i.IFiles,
	messages i.IMessages,
) (*Task, types.ScheduleResponseStatus) {
	scheduler.lock.AssertLocked()

	var extractedTask *Task

	// First look in memory
	{
		var indexToRemove int
		for index, currTask := range scheduler.tasks {
			if currTask.Computed.Id == taskId {
				if currTask.Dynamic.IsDeactivated {
					indexToRemove = index
					extractedTask = currTask
					break
				} else {
					return nil, types.ScheduleResponseStatusTaskNotDeactivated
				}
			}
		}

		if extractedTask != nil {
			// Remove task from the list and return it
			newCount := len(scheduler.tasks) - 1
			scheduler.tasks[indexToRemove] = scheduler.tasks[newCount]
			scheduler.tasks = scheduler.tasks[:newCount]
			return extractedTask, types.ScheduleResponseStatusSuccess
		}

	}

	// If we're here, we didn't find a task in memory. Look in ndjson file. We'll have to remove extracted task from the file,
	// so we're also opening temporary output file and writing to it all lines except for the removed task. Then we'll copy
	// it to be the new ndjson file.
	{
		inputFilePath := files.GetDeactivatedTasksFile()
		inputFile, err := os.OpenFile(inputFilePath, os.O_RDONLY, 0644)
		if err != nil {
			messages.AddF(i.BackendMessageError, nil, "Failed reading trimmed tasks: %s", err.Error())
			return nil, types.ScheduleResponseStatusTaskNotFound
		}
		defer inputFile.Close()

		outputFile, err := files.GetTmpFile()
		outputFilePath := outputFile.Name()
		defer os.Remove(outputFilePath)
		defer outputFile.Close()
		if err != nil {
			messages.AddF(i.BackendMessageError, nil, "Failed opening tmp file: %s", err.Error())
			return nil, types.ScheduleResponseStatusTaskNotFound
		}

		scanner := bufio.NewScanner(inputFile)
		for scanner.Scan() {
			line := scanner.Bytes()
			var currentTask Task
			err := json.Unmarshal(line, &currentTask)
			if err != nil {
				messages.AddF(i.BackendMessageError, nil, "Failed decoding a task from %s: %s", inputFilePath, err.Error())
				continue
			}

			if extractedTask == nil && currentTask.Computed.Id == taskId {
				extractedTask = &currentTask
			} else {
				line = append(line, '\n')
				if err := common.WriteBytesToWriter(outputFile, line); err != nil {
					messages.AddF(i.BackendMessageError, nil, "Failed writing to tmp file")
					return nil, types.ScheduleResponseStatusTaskNotFound
				}
			}
		}

		if extractedTask != nil {
			inputFile.Close()
			outputFile.Close()
			if err := common.CopyFile(outputFilePath, inputFilePath); err != nil {
				messages.AddF(i.BackendMessageError, nil, "Failed copying tmp file to ndjson")
				return nil, types.ScheduleResponseStatusTaskNotFound
			}

			return extractedTask, types.ScheduleResponseStatusSuccess
		}
	}

	// If we're here, we didn't find the task neither in memory nor in ndjson file
	return nil, types.ScheduleResponseStatusTaskNotFound
}

func (scheduler *Scheduler) CheckForTaskConflict(newTask *Task) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	for _, currTask := range scheduler.tasks {
		if !currTask.Dynamic.IsDeactivated {
			if currTask.Computed.Hash == newTask.Computed.Hash {
				return types.ScheduleResponseStatusAlreadyRunning
			}
			if currTask.FriendlyName != "" && currTask.Computed.NameDisplayHash == newTask.Computed.NameDisplayHash {
				return types.ScheduleResponseStatusNameDisplayAlreadyRunning
			}
		}
	}

	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) CheckForDisplay(
	newTask *Task,
	displays i.IDisplays,
	goroutines i.IGoroutines,
	messages i.IMessages,
) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	switch newTask.Display.Type {
	case types.DisplaySelectionTypeHeadless:
	case types.DisplaySelectionTypeXorg, types.DisplaySelectionTypeWayland:
		err := displays.InitDisplay(newTask.Display, scheduler, goroutines, messages)
		if err != nil {
			return types.ScheduleResponseStatusInvalidDisplay
		}
	default:
		messages.Add(i.BackendMessageError, newTask, "Invalid display type")
	}

	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) TryScheduleTask(
	newTask *Task,
	files i.IFiles,
	displays i.IDisplays,
	goroutines i.IGoroutines,
	messages i.IMessages,
) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	// Calculate internal properties
	newTask.Init(scheduler.currentId, files.GetTaskLogFile(scheduler.currentId))
	scheduler.currentId++

	// Do not schedule, if a similar task is already running
	if status := scheduler.CheckForTaskConflict(newTask); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Ensure display is correct
	if status := scheduler.CheckForDisplay(newTask, displays, goroutines, messages); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Schedule
	scheduler.tasks = append(scheduler.tasks, newTask)
	goroutines.StartGoroutine(func() {
		ExecuteTask(newTask, &scheduler.lock, files, goroutines, messages)
	})
	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) TryRescheduleTask(
	newTask *Task,
	files i.IFiles,
	displays i.IDisplays,
	goroutines i.IGoroutines,
	messages i.IMessages,
) types.ScheduleResponseStatus {
	scheduler.lock.AssertLocked()

	// Calculate internal properties
	newTask.Init(newTask.Computed.Id, files.GetTaskLogFile(newTask.Computed.Id))

	// Do not schedule, if a similar task is already running
	if status := scheduler.CheckForTaskConflict(newTask); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Ensure display is correct
	if status := scheduler.CheckForDisplay(newTask, displays, goroutines, messages); status != types.ScheduleResponseStatusSuccess {
		return status
	}

	// Schedule
	scheduler.tasks = append(scheduler.tasks, newTask)
	goroutines.StartGoroutine(func() {
		ExecuteTask(newTask, &scheduler.lock, files, goroutines, messages)
	})
	return types.ScheduleResponseStatusSuccess
}

func (scheduler *Scheduler) StopTasksByDisplay(display types.DisplaySelection) {
	scheduler.lock.AssertLocked()

	for _, currTask := range scheduler.tasks {
		if currTask.Display == display {
			currTask.Channels.StopChannel <- fmt.Sprintf("stopping tasks on %v display %v", display.Type.String(), display.Name)
		}
	}
}
