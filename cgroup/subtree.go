package cgroup

import (
	"os"
	"path/filepath"
)

func HasSubtree(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(CgroupPath, path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CreateSubtree(path string) error {
	return os.MkdirAll(filepath.Join(CgroupPath, path), 0700)
}

func DeleteSubtree(path string) error {
	return os.RemoveAll(filepath.Join(CgroupPath, path))
}
