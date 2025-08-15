package backend

import (
	"bufio"
	"fmt"
	"io"
	"os"
	i "spieven/backend/interfaces"
	"spieven/common"
	"sync"
)

type LogMessage struct {
	msg          string
	isDiagnostic bool
	isSeparator  bool
	isStop       bool
}

func diagnosticMessage(content string, isSeparator bool) LogMessage {
	return LogMessage{
		msg:          content,
		isDiagnostic: true,
		isSeparator:  isSeparator,
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

type FileLogger struct {
	files                 i.IFiles
	goroutines            i.IGoroutines
	channel               chan LogMessage
	errorChannel          chan error
	stdoutFilePathChannel chan string
	waitGroup             sync.WaitGroup
	outFilePath           string
	taskId                int
	captureStdout         bool

	_ common.NoCopy
}

func CreateFileLogger(files i.IFiles, goroutines i.IGoroutines, taskId int, captureStdout bool) FileLogger {
	return FileLogger{
		files:                 files,
		goroutines:            goroutines,
		channel:               make(chan LogMessage),
		errorChannel:          make(chan error, 1),
		stdoutFilePathChannel: make(chan string),
		waitGroup:             sync.WaitGroup{},
		taskId:                taskId,
		captureStdout:         captureStdout,
	}

}

func (log *FileLogger) run() error {
	// Setup WaitGroup to notify caller that we've finished
	log.waitGroup.Add(1)
	defer log.waitGroup.Done()

	// Open task file for writing
	taskFile, err := os.Create(log.files.GetTaskLogFile(log.taskId))
	if err != nil {
		return fmt.Errorf("failed opening task log file")
	}

	// Open stdout file for writing. We're going to reopen it as soon as task execution ends, so each execution gets
	// its own stdout file.
	taskExecutionId := 0
	var stdoutFile *os.File
	if log.captureStdout {
		stdoutFile, err = os.Create(log.files.GetStdoutLogFile(log.taskId, taskExecutionId))
		if err != nil {
			return fmt.Errorf("failed opening stdout file")
		}
	}

	log.goroutines.StartGoroutine(func() {
		// Main loop
		var loggingErr error
		for {
			// Query message from channel.
			message := <-log.channel

			// Handle stop message, meaning we should exit the logger. This is typically sent when the task gets deactivated.
			if message.isStop {
				break
			}

			// Handle separator message, meaning the task execution ended. String content of separator messages is ignored.
			if message.isSeparator {
				// Stdout file - close it and open a new one
				taskExecutionId++
				newStdoutFilePath := log.files.GetStdoutLogFile(log.taskId, taskExecutionId)
				oldStdoutFilePath := ""
				oldStdoutFilePath, loggingErr = log.FinalizeStdout(&stdoutFile, &newStdoutFilePath)
				log.stdoutFilePathChannel <- oldStdoutFilePath
				if loggingErr != nil {
					break
				}

				// Task file - write separator lines
				if oldStdoutFilePath != "" {
					stdoutMsg := fmt.Sprintf("Stdout written to %v", oldStdoutFilePath)
					common.WriteStringToWriter(taskFile, log.WrapWithDiagnosticDecorations(&stdoutMsg))
				}
				common.WriteStringToWriter(taskFile, "\n\n\n")

				continue
			}

			// Stdout file
			if !message.isDiagnostic && stdoutFile != nil {
				loggingErr = common.WriteStringToWriter(stdoutFile, message.msg)
				if loggingErr != nil {
					break
				}
			}

			// Task file
			taskFileContent := message.msg
			if message.isDiagnostic {
				taskFileContent = log.WrapWithDiagnosticDecorations(&taskFileContent)
			}
			loggingErr = common.WriteStringToWriter(taskFile, taskFileContent)
			if loggingErr != nil {
				break
			}
		}

		// Propagate any errors to a caller via error channel
		if loggingErr != nil {
			log.errorChannel <- fmt.Errorf("failed writing to %v", log.outFilePath)
		}

		// Cleanup files
		taskFile.Close()
		log.FinalizeStdout(&stdoutFile, nil)
	})

	return nil
}

func (log *FileLogger) WrapWithDiagnosticDecorations(message *string) string {
	return fmt.Sprintf("--------------------- %v ---------------------\n", *message)
}

func (log *FileLogger) FinalizeStdout(stdoutFile **os.File, newPath *string) (string, error) {
	if !log.captureStdout {
		return "", nil
	}

	oldPath := (**stdoutFile).Name()
	(**stdoutFile).Close()
	*stdoutFile = nil

	if newPath != nil {
		var err error
		*stdoutFile, err = os.Create(*newPath)
		if err != nil {
			return oldPath, fmt.Errorf("failed opening stdout file")
		}
	}

	return oldPath, nil
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
