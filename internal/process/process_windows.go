package process

import "os"

func isAlive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}

func terminateProcess(proc *os.Process) error {
	return proc.Kill()
}
