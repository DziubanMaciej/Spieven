package frontend

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"supervisor/common"
	"sync/atomic"
	"time"
)

func WatchFile(taskId int, filePath string, stopFlag *atomic.Int32) error {

	file, err := common.OpenFileWithTimeout(filePath, os.O_RDONLY, 0644, 2*time.Second)
	if err != nil {
		return fmt.Errorf("could not open log file. Try running Spieven peek %v", taskId)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// Print current contents
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // reached end of current content
			} else {
				return err // other error
			}
		}
		fmt.Print(line)
	}

	// Continuously wait for new data
	for {
		if stopFlag.Load() != 0 {
			return nil
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Millisecond * 100)
			} else {
				return err
			}
		}
		fmt.Print(line)
	}
}
