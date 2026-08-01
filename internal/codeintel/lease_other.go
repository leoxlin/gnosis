//go:build !linux

package codeintel

import (
	"os"
)

type writerLease struct {
	file *os.File
	path string
}

func acquireLease(path string) (*writerLease, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	return &writerLease{file: file, path: path}, nil
}

func (lease *writerLease) Close() error {
	err := lease.file.Close()
	removeErr := os.Remove(lease.path)
	if err != nil {
		return err
	}
	return removeErr
}
