package scheduler

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
	isStderr     bool
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

func outMessage(message string, isStderr bool) LogMessage {
	return LogMessage{
		msg:      fmt.Sprintf("%v\n", message),
		isStderr: isStderr,
	}
}

type LogResponse struct {
	err            error
	stdoutFilePath string
	stderrFilePath string
}

type FileLogger struct {
	files         i.IFiles
	goroutines    i.IGoroutines
	channel       chan LogMessage  // input channel for incoming messages
	outChannel    chan LogResponse // output channel for errors or diagnostics
	waitGroup     sync.WaitGroup
	taskId        int
	captureStdout bool
	captureStderr bool

	_ common.NoCopy
}

func CreateFileLogger(files i.IFiles, goroutines i.IGoroutines, taskId int, captureStdout bool, captureStderr bool) FileLogger {
	return FileLogger{
		files:         files,
		goroutines:    goroutines,
		channel:       make(chan LogMessage),
		outChannel:    make(chan LogResponse, 1),
		waitGroup:     sync.WaitGroup{},
		taskId:        taskId,
		captureStdout: captureStdout,
		captureStderr: captureStderr,
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

	// Open stdout/stderr files for writing. We're going to reopen them as soon as task execution ends, so each execution gets
	// its own stdout/stderr files.
	taskExecutionId := 0
	stdoutFilePath, stderrFilePath := log.files.GetStdoutStderrLogFiles(log.taskId, taskExecutionId)
	var stdoutFile *os.File
	var stderrFile *os.File
	if log.captureStdout {
		stdoutFile, err = os.Create(stdoutFilePath)
		if err != nil {
			return fmt.Errorf("failed opening stdout file")
		}
	}
	if log.captureStderr {
		stderrFile, err = os.Create(stderrFilePath)
		if err != nil {
			if stdoutFile != nil {
				stdoutFile.Close()
			}
			return fmt.Errorf("failed opening stderr file")
		}
	}

	// Start the main loop in a goroutine. It will run until it receives a stop message.
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
				// Reopen stdout and stderr files for the next execution with an incremented execution ID.
				taskExecutionId++
				var oldStdoutFilePath, oldStderrFilePath string
				newStdoutFilePath, newStderrFilePath := log.files.GetStdoutStderrLogFiles(log.taskId, taskExecutionId)
				oldStdoutFilePath, oldStderrFilePath, loggingErr = log.FinalizeStdoutAndStderr(&stdoutFile, &stderrFile, &newStdoutFilePath, &newStderrFilePath)

				// Notify the caller that the stdout and stderr files have been closed.
				log.outChannel <- LogResponse{
					stdoutFilePath: oldStdoutFilePath,
					stderrFilePath: oldStderrFilePath,
				}

				if loggingErr != nil {
					break
				}

				// Task file - write separator lines
				if oldStdoutFilePath != "" {
					stdoutMsg := fmt.Sprintf("Stdout written to %v", oldStdoutFilePath)
					common.WriteStringToWriter(taskFile, log.WrapWithDiagnosticDecorations(&stdoutMsg))
				}
				if oldStderrFilePath != "" {
					stderrMsg := fmt.Sprintf("Stderr written to %v", oldStderrFilePath)
					common.WriteStringToWriter(taskFile, log.WrapWithDiagnosticDecorations(&stderrMsg))
				}
				common.WriteStringToWriter(taskFile, "\n\n\n")

				continue
			}

			// Stdout file
			if !message.isDiagnostic {
				if message.isStderr && stderrFile != nil {
					loggingErr = common.WriteStringToWriter(stderrFile, message.msg)
				} else if stdoutFile != nil {
					loggingErr = common.WriteStringToWriter(stdoutFile, message.msg)
				}
				if loggingErr != nil {
					fmt.Printf("BBB for isStderr=%v  err: %v\n", message.isStderr, loggingErr.Error())
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

		// Propagate any errors to the caller via output channel
		if loggingErr != nil {
			log.outChannel <- LogResponse{
				err: fmt.Errorf("failed writing to per-task stdout/stderr files"),
			}
		}

		// Cleanup files
		taskFile.Close()
		log.FinalizeStdoutAndStderr(&stdoutFile, &stderrFile, nil, nil)
	})

	return nil
}

func (log *FileLogger) WrapWithDiagnosticDecorations(message *string) string {
	return fmt.Sprintf("--------------------- %v ---------------------\n", *message)
}

func (log *FileLogger) FinalizeStdoutAndStderr(stdoutFile **os.File, stderrFile **os.File, newStdoutPath *string, newStderrPath *string) (string, string, error) {
	oldStdoutFilePath := ""
	oldStderrFilePath := ""
	var err error

	if (*stdoutFile) != nil {
		oldStdoutFilePath = (**stdoutFile).Name()
		(**stdoutFile).Close()
		*stdoutFile = nil

		if newStdoutPath != nil {
			*stdoutFile, err = os.Create(*newStdoutPath)
			if err != nil {
				err = fmt.Errorf("failed creating new stdout file %v", *newStdoutPath)
			}
		}
	}

	if (*stderrFile) != nil {
		oldStderrFilePath = (**stderrFile).Name()
		(**stderrFile).Close()
		*stderrFile = nil

		if newStderrPath != nil {
			*stderrFile, err = os.Create(*newStderrPath)
			if err != nil {
				err = fmt.Errorf("failed creating new stderr file %v", *newStderrPath)
			}
		}
	}

	if err != nil {
		if stdoutFile != nil {
			(*stdoutFile).Close()
			*stdoutFile = nil
		}
		if stderrFile != nil {
			(*stderrFile).Close()
			*stderrFile = nil
		}
	}

	return oldStdoutFilePath, oldStderrFilePath, err
}

func (log *FileLogger) stop() {
	log.channel <- stopMessage()
	log.waitGroup.Wait()
}

func (log *FileLogger) streamOutput(reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		t := scanner.Text()
		log.channel <- outMessage(t, isStderr)
	}

	if err := scanner.Err(); err != nil {
		panic("Classic \"This will never happen moment\". See you in a bit :)")
	}
}
