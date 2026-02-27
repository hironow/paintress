//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

var shutdownSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
