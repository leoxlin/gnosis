//go:build linux

package codeintel

import (
	"os"
	"syscall"
)

type writerLease struct{ file *os.File }

func acquireLease(path string) (*writerLease, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return &writerLease{file: file}, nil
}

func (lease *writerLease) Close() error {
	if err := syscall.Flock(int(lease.file.Fd()), syscall.LOCK_UN); err != nil {
		lease.file.Close()
		return err
	}
	return lease.file.Close()
}
