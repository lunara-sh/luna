package process

import (
	"os"
	"syscall"
)

func isAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func terminateProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGALRM)
}
