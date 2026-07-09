package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func writeGeneratedFile(path string, content []byte, overwrite bool) (bool, error) {
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if !overwrite || bytes.Equal(existing, content) {
			return false, nil
		}
	case !os.IsNotExist(err):
		return false, err
	}

	if err := atomicWriteFile(path, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func atomicWriteFile(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(mode); err != nil {
		return err
	}
	if _, err := temp.Write(content); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
