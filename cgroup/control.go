package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

func AddProcessToSubtree(subtree string, pid int) error {
	file, err := os.OpenFile(filepath.Join(CgroupPath, subtree, "cgroup.procs"), os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(file, "%d", pid)
	return err
}
