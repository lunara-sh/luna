//go:build !windows

package process

import (
	"os"
	"syscall"
	"time"
)

func isAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// terminateProcess attempts a graceful shutdown with SIGTERM, then escalates
// to SIGKILL if the process is still alive after a short grace period.
func terminateProcess(proc *os.Process) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// Wait up to ~5s for the process to exit gracefully.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !isAlive(proc.Pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Still alive — force kill.
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		if !isAlive(proc.Pid) {
			return nil
		}
		return err
	}
	return nil
}
