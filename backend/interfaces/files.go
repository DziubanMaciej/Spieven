package interfaces

import "os"

type IFiles interface {
	GetTmpFile() (*os.File, error)
	GetDeactivatedTasksFile() string
	GetTaskLogFile(taskId int) string
	GetStdoutLogFile(taskId int, executionId int) string
	GetBackendMessagesLogFile() string
}
