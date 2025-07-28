package backend

import (
	"fmt"
	"os"
	"supervisor/common"
	"sync"
	"time"
)

type BackendMessageSeverity int

const (
	BackendMessageInfo BackendMessageSeverity = iota
	BackendMessageError
)

// BackendMessage is a description of a failure encountered by the backend that is to be stored for later retrieval by
// an appropriate frontend command.
type BackendMessage struct {
	date     time.Time
	severity BackendMessageSeverity
	content  string
	task     *Task
}

func (msg *BackendMessage) String() string {
	result := msg.content

	var severity string

	switch msg.severity {
	case BackendMessageInfo:
		severity = " INFO"
	case BackendMessageError:
		severity = "ERROR"
	default:
		severity = "     " // Should not happen, but let's handle it gracefully
	}

	date := msg.date.Format("2006-01-02 15-04-05")
	taskLabel := ""
	if msg.task != nil {
		taskLabel = fmt.Sprintf(" (%s)", msg.task.Computed.LogLabel)
	}
	result = fmt.Sprintf("[%v][%v] %v%v", severity, date, result, taskLabel)

	return result
}

// BackendMessages stores instances of BackendMessage and exposes methods to retrieve and manage them.
type BackendMessages struct {
	messages []BackendMessage
	logFile  *os.File
	lock     sync.Mutex

	_ common.NoCopy
}

func CreateBackendMessages(logFilePath string) (*BackendMessages, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_EXCL|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &BackendMessages{
		logFile: logFile,
	}, nil
}

func (messages *BackendMessages) Cleanup() {
	if messages.logFile != nil {
		messages.logFile.Close()
		messages.logFile = nil
	}
}

func (messages *BackendMessages) String() string {
	messages.lock.Lock()
	defer messages.lock.Unlock()

	var result string
	for _, BackendMessage := range messages.messages {
		result += BackendMessage.content
	}
	return result
}

func (messages *BackendMessages) Add(severity BackendMessageSeverity, task *Task, content string) {
	msg := BackendMessage{
		date:     time.Now(),
		severity: severity,
		content:  content,
		task:     task,
	}

	fmt.Println(msg.String())

	messages.lock.Lock()
	messages.messages = append(messages.messages, msg)

	if messages.logFile != nil {
		msgWithNewline := msg.String() + "\n"
		err := common.WriteBytesToWriter(messages.logFile, []byte(msgWithNewline))
		if err != nil {
			messages.logFile.Close()
			messages.logFile = nil
		}
	}

	messages.lock.Unlock()
}

func (messages *BackendMessages) AddF(severity BackendMessageSeverity, task *Task, format string, args ...any) {
	content := fmt.Sprintf(format, args...)
	messages.Add(severity, task, content)
}

func (messages *BackendMessages) Trim(maxAge time.Duration) {
	messages.lock.Lock()
	defer messages.lock.Unlock()

	now := time.Now()

	var newBackendMessages []BackendMessage
	for _, BackendMessage := range messages.messages {
		deadline := BackendMessage.date.Add(maxAge)
		if now.Before(deadline) {
			newBackendMessages = append(newBackendMessages, BackendMessage)
		}
	}

	messages.messages = newBackendMessages
}
