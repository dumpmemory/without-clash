package main

import "io"

func closeSilent(closer io.Closer) {
	_ = closer.Close()
}
