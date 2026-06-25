package process

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func IsPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

func ReadPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pid in %s: %s", path, pidStr)
	}
	return pid, nil
}

func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	if runtime.GOOS == "windows" {
		_, err := os.FindProcess(pid)
		return err == nil
	}

	return isAlive(pid)
}

func KillByPidFile(pidFile string) error {
	pid, err := ReadPidFile(pidFile)
	if err != nil {
		return fmt.Errorf("cannot read pid file: %w", err)
	}
	return KillPid(pid)
}

func KillPid(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return terminateProcess(proc)
}

func FindProcessByName(name string) ([]int, error) {
	if runtime.GOOS == "windows" {
		return findByNameWindows(name)
	}
	return findByNameUnix(name)
}

func findByNameUnix(name string) ([]int, error) {
	out, err := exec.Command("pgrep", "-x", name).Output()
	if err != nil {
		return nil, nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if pid, err := strconv.Atoi(line); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func findByNameWindows(name string) ([]int, error) {
	out, err := exec.Command(
		"tasklist",
		"/FI", fmt.Sprintf("IMAGENAME eq %s", name),
		"/FO", "CSV",
		"/NH",
	).Output()
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			pidStr := strings.Trim(parts[1], `" `)
			if pid, err := strconv.Atoi(pidStr); err == nil {
				pids = append(pids, pid)
			}
		}
	}
	return pids, nil
}
