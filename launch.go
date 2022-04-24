package main

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

const launchArg0 = "without-clash"

var (
	proxyEnvs = map[string]struct{}{
		"all_proxy":   {},
		"http_proxy":  {},
		"https_proxy": {},
		"ftp_proxy":   {},
		"no_proxy":    {},
	}
)

func launch() {
	if len(os.Args) <= 1 {
		println("Usage: without-clash <command...>")

		runtime.Goexit()
	}

	commandPath, err := exec.LookPath(os.Args[1])
	if err != nil {
		abort("Command %s not found: ", os.Args[1], err.Error())
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

	conn, err := net.Dial("unix", "@"+daemonArg0)
	if err != nil {
		abort("Dial to daemon: %s", err.Error())
	}

	_, err = conn.Write([]byte{0})
	if err != nil {
		abort("Write to daemon: %s", err.Error())
	}

	_, err = conn.Read([]byte{0})
	if err != nil {
		abort("Receive reply: %s", err.Error())
	}

	err = unix.Exec(commandPath, os.Args[1:], filteredEnv)
	if err != nil {
		abort("Exec '%s': %s", strings.Join(os.Args[1:], " "), err.Error())
	}
}
