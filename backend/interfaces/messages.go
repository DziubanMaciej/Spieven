package interfaces

type MessageSeverity int

const (
	BackendMessageInfo MessageSeverity = iota
	BackendMessageError
)

type IMessages interface {
	Add(severity MessageSeverity, task ITask, content string)
	AddF(severity MessageSeverity, task ITask, format string, args ...any)
}
