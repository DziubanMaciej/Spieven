package interfaces

import "os"

type IFiles interface {
	GetTmpFile() (*os.File, error)
	GetDeactivatedTasksFile() string
	GetTaskLogFile(taskId int) string
	GetStdoutStderrLogFiles(taskId int, executionId int) (string, string)
	GetBackendMessagesLogFile() string
}
