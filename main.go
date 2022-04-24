package main

import (
	"os"
	"path/filepath"
)

var exitCode = 0

func main() {
	defer os.Exit(exitCode)

	switch filepath.Base(os.Args[0]) {
	case daemonArg0:
		daemon()
	case launchArg0:
		launch()
	default:
		abort("Unsupported applet: %s", os.Args[0])
	}
}
