package common

import (
	"hash/fnv"
	"io"
	"os"
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
