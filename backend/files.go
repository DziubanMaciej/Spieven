package backend

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"supervisor/common"
)

type FilePathProvider struct {
	CacheDir               string
	TmpDir                 string
	TaskLogsDir            string
	DeactivatedTasksFile   string
	BackendMessagesLogFile string

	_ common.NoCopy
}

func CreateFilePathProvider() (*FilePathProvider, error) {
	homeDir, found := os.LookupEnv("HOME")
	if !found {
		return nil, errors.New("failed to read HOME env var")
	}

	cacheDir := path.Join(homeDir, ".cache", "Spieven")
	err := EnsureDirExistsAndIsEmpty(cacheDir)
	if err != nil {
		return nil, err
	}

	tmpDir := path.Join(cacheDir, "tmp")
	err = EnsureDirExistsAndIsEmpty(tmpDir)
	if err != nil {
		return nil, err
	}

	taskLogsDir := path.Join(cacheDir, "tasks")
	err = EnsureDirExistsAndIsEmpty(taskLogsDir)
	if err != nil {
		return nil, err
	}

	deactivatedTasksFile := path.Join(cacheDir, "deactivatedTasks.ndjson")
	err = EnsureFileExistsAndIsEmpty(deactivatedTasksFile)
	if err != nil {
		return nil, err
	}

	backendMessagesLogFile := path.Join(cacheDir, "backend.log")
	err = EnsureFileExistsAndIsEmpty(backendMessagesLogFile)
	if err != nil {
		return nil, err
	}

	return &FilePathProvider{
		CacheDir:               cacheDir,
		TmpDir:                 tmpDir,
		TaskLogsDir:            taskLogsDir,
		DeactivatedTasksFile:   deactivatedTasksFile,
		BackendMessagesLogFile: backendMessagesLogFile,
	}, nil
}

func (files *FilePathProvider) GetTmpFile() (*os.File, error) {
	return os.CreateTemp(files.TmpDir, "")
}

func (files *FilePathProvider) GetDeactivatedTasksFile() string {
	return files.DeactivatedTasksFile
}

func (files *FilePathProvider) GetTaskLogFile(taskId int) string {
	fileName := fmt.Sprintf("task_%03d.log", taskId)
	return path.Join(files.TaskLogsDir, fileName)
}

func (files *FilePathProvider) GetStdoutLogFile(taskId int, executionId int) string {
	fileName := fmt.Sprintf("task_%03d_stdout_%03d.log", taskId, executionId)
	return path.Join(files.TaskLogsDir, fileName)
}

func (files *FilePathProvider) GetBackendMessagesLogFile() string {
	return files.BackendMessagesLogFile
}

func (files *FilePathProvider) Cleanup() error {
	return os.RemoveAll(files.CacheDir)
}

func EnsureDirExistsAndIsEmpty(dir string) error {
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		} else {
			return err
		}
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is a file, while directory was expected", dir)
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range dirEntries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

func EnsureFileExistsAndIsEmpty(file string) error {
	fileInfo, err := os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("%s exists, but it's not a file", file)
		}
	}

	fileHandle, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	fileHandle.Close()
	return nil
}
