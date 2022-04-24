package cgroup

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func IsVersion2() bool {
	mounts, err := os.OpenFile("/proc/self/mounts", os.O_RDONLY, 0)
	if err != nil {
		panic(err.Error())
	}
	defer mounts.Close()

	reader := bufio.NewReader(mounts)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err.Error())
		}

		fields := strings.Fields(string(line))
		if len(fields) < 3 {
			continue
		}

		target := fields[1]
		if target != CgroupPath {
			continue
		}

		source := fields[0]
		fs := fields[2]
		if source == "cgroup2" && fs == "cgroup2" {
			return true
		}
	}

	return false
}
