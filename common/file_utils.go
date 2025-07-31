package common

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"time"
)

func CalculateSpievenFileHash() (uint64, error) {
	path := os.Args[0]
	return CalculateFileHash(path)
}

func CalculateFileHash(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	h := fnv.New64a()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

func WriteBytesToWriter(writer io.Writer, value []byte) error {
	written := 0
	for written < len(value) {
		writtenThisIteration, err := writer.Write(value[written:])
		if err != nil {
			return err
		}

		written += writtenThisIteration
	}
	return nil
}

func OpenFileWithTimeout(filePath string, flag int, perm os.FileMode, timeout time.Duration) (file *os.File, err error) {
	iterations := 5
	deadline := time.Now()
	timeoutPerIteration := timeout / time.Duration(iterations)

	for i := 0; i < iterations; i++ {
		fmt.Printf("Iter %v (timeoutPerIteration=%v)\n", i, timeoutPerIteration)
		file, err = os.OpenFile(filePath, flag, perm)
		if err == nil {
			break
		} else {
			now := time.Now()
			deadline = deadline.Add(timeoutPerIteration)
			if now.Before(deadline) {
				time.Sleep(deadline.Sub(now))
			}
		}

	}
	return
}
