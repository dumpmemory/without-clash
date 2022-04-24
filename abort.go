package main

import (
	"fmt"
	"os"
	"runtime"
)

func abort(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)

	exitCode = 1

	runtime.Goexit()
}
