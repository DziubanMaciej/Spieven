package backend

import (
	"fmt"
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
	// TODO add process description and find some way to log nicely. Maybe allow setting "friendlyName" for each process?
}

func (msg *BackendMessage) String() string {
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

	return fmt.Sprintf("[%v][%v] %v\n", severity, date, msg.content)
}

// BackendMessages stores instances of BackendMessage and exposes methods to retrieve and manage them.
type BackendMessages struct {
	messages []BackendMessage
	lock     sync.Mutex
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

func (messages *BackendMessages) Add(severity BackendMessageSeverity, content string) BackendMessage {
	msg := BackendMessage{
		date:     time.Now(),
		severity: severity,
		content:  content,
	}

	fmt.Print(msg.String())

	messages.lock.Lock()
	messages.messages = append(messages.messages, msg)
	messages.lock.Unlock()

	return msg
}

func (messages *BackendMessages) AddF(severity BackendMessageSeverity, format string, args ...any) BackendMessage {
	content := fmt.Sprintf(format, args...)
	return messages.Add(severity, content)
}

func (messages *BackendMessages) Trim(maxAge time.Duration) {
	messages.lock.Lock()
	defer messages.lock.Unlock()

	now := time.Now()

	var newBackendMessages []BackendMessage
	for _, BackendMessage := range messages.messages {
		age := now.Sub(BackendMessage.date)
		if age < maxAge {
			newBackendMessages = append(newBackendMessages, BackendMessage)
		}
	}

	messages.messages = newBackendMessages
}
