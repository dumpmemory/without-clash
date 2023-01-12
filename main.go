package main

import (
	"flag"
	"os"
)

var (
	argDaemon = flag.Bool("daemon", false, "Run daemon.")
	argFork   = flag.Bool("fork", false, "Fork child process to avoid cgroup conflict.")
)

func main() {
	flag.Parse()

	if *argDaemon {
		err := runDaemon()
		if err != nil {
			println("Without clash daemon:", err.Error())

			os.Exit(1)
		}

		os.Exit(0)
	}

	commands := flag.Args()
	if len(commands) == 0 {
		flag.Usage()

		os.Exit(255)
	}

	err := runExec(*argFork, commands)
	if err != nil {
		println("Without clash exec:", err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}
