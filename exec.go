package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	proxyEnvs = map[string]struct{}{
		"all_proxy":   {},
		"http_proxy":  {},
		"https_proxy": {},
		"ftp_proxy":   {},
		"no_proxy":    {},
	}
)

func runExec(shouldFork bool, commands []string) error {
	if shouldFork {
		myExecutable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve self executable: %w", err)
		}

		args := []string{myExecutable}
		args = append(args, commands...)

		cmd := &exec.Cmd{
			Path:   myExecutable,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		err = cmd.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return fmt.Errorf("launch child process: %w", err)
			}
		}

		os.Exit(cmd.ProcessState.ExitCode())
	}

	executablePath, err := exec.LookPath(commands[0])
	if err != nil {
		return fmt.Errorf("lookup command %s: %w", commands[0], err)
	}

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: listenPath, Net: "unix"})
	if err != nil {
		return fmt.Errorf("dial to daemon: %w", err)
	}

	_, err = conn.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	_, err = conn.Read([]byte{0})
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}

	var filteredEnv []string
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			if _, ok := proxyEnvs[strings.ToLower(kv[0])]; ok {
				continue
			}
		}
		filteredEnv = append(filteredEnv, env)
	}

	err = unix.Exec(executablePath, commands, filteredEnv)
	if err != nil {
		return fmt.Errorf("exec command '%s': %w", strings.Join(commands, " "), err)
	}

	panic("unreachable")
}
