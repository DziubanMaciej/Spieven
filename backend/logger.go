package backend

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type LogMessage struct {
	msg          string
	isDiagnostic bool
	isStop       bool
}

func diagnosticMessageF(format string, a ...any) LogMessage {
	return LogMessage{
		msg:          fmt.Sprintf(format, a...),
		isDiagnostic: true,
	}
}

func stopMessage() LogMessage {
	return LogMessage{
		isStop: true,
	}
}

func stdoutMessage(message string) LogMessage {
	return LogMessage{
		msg: fmt.Sprintf("%v\n", message),
	}
}

func emptyLinesMessage(lines int) LogMessage {
	return LogMessage{
		msg: strings.Repeat("\n", lines),
	}
}

type FileLogger struct {
	channel      chan LogMessage
	errorChannel chan error
	waitGroup    sync.WaitGroup
	outFilePath  string
}

func CreateFileLogger(outFilePath string) FileLogger {
	return FileLogger{
		channel:      make(chan LogMessage),
		errorChannel: make(chan error, 1),
		waitGroup:    sync.WaitGroup{},
		outFilePath:  outFilePath,
	}

}

func (log *FileLogger) run() error {
	// Setup WaitGroup to notify caller that we've finished
	log.waitGroup.Add(1)
	defer log.waitGroup.Done()

	// Open file for writing
	outFile, err := os.Create(log.outFilePath)
	if err != nil {
		return fmt.Errorf("failed opening %v", log.outFilePath)
	}

	go func() {
		defer outFile.Close()
		for {
			// Query message from channel.
			message := <-log.channel
			if message.isStop {
				break
			}

			// Prepare the message
			chunk := message.msg
			if message.isDiagnostic {
				chunk = fmt.Sprintf("--------------------- %v ---------------------\n", chunk)
			}

			// Write the message
			written := 0
			for written < len(chunk) {
				writtenThisIteration, err := outFile.WriteString(chunk[written:])
				if err != nil {
					log.errorChannel <- fmt.Errorf("failed writing to %v", log.outFilePath)
					return
				}

				written += writtenThisIteration
			}

		}
	}()

	return nil
}

func (log *FileLogger) stop() {
	log.channel <- stopMessage()
	log.waitGroup.Wait()
}

func (log *FileLogger) streamOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		t := scanner.Text()
		log.channel <- stdoutMessage(t)
	}

	if err := scanner.Err(); err != nil {
		panic("Classic \"This will never happen moment\". See you in a bit :)")
	}
}
