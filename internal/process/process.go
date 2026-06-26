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
	// Some pid files (e.g. PostgreSQL's postmaster.pid) store the PID on the
	// first line followed by additional metadata, so only parse the first line.
	content := strings.TrimSpace(string(data))
	firstLine := content
	if idx := strings.IndexAny(content, "\r\n"); idx >= 0 {
		firstLine = strings.TrimSpace(content[:idx])
	}
	pid, err := strconv.Atoi(firstLine)
	if err != nil {
		return 0, fmt.Errorf("invalid pid in %s: %s", path, firstLine)
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
	// First try an exact process-name match (handles caddy, mysqld, etc.).
	if pids := pgrep("-x", name); len(pids) > 0 {
		return pids, nil
	}
	// Fall back to a full command-line match. Some daemons rewrite their
	// process name (e.g. PHP-FPM workers show up as "php-fpm: pool www"),
	// so an exact match misses them while a full-line match catches them.
	return pgrep("-f", name), nil
}

func pgrep(flag, name string) []int {
	out, err := exec.Command("pgrep", flag, name).Output()
	if err != nil {
		return nil
	}
	self := os.Getpid()
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if pid, err := strconv.Atoi(line); err == nil && pid != self {
			pids = append(pids, pid)
		}
	}
	return pids
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
