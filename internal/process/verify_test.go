package process

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestHelperProcess is not a real test. When invoked with GO_DAEMON_HELPER=1 it
// behaves like a daemon that ignores SIGALRM but exits cleanly on SIGTERM,
// mirroring Caddy / PHP-FPM / MySQL. This makes the SIGALRM behavior explicit
// and portable instead of depending on shell trap semantics.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_DAEMON_HELPER") != "1" {
		return
	}
	signal.Ignore(syscall.SIGALRM)
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM)
	<-term
	os.Exit(0)
}

func startDaemon(t *testing.T) (*exec.Cmd, chan struct{}) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_DAEMON_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	time.Sleep(300 * time.Millisecond)
	if !IsProcessRunning(cmd.Process.Pid) {
		t.Fatalf("daemon should be running (pid %d)", cmd.Process.Pid)
	}
	return cmd, exited
}

// Regression test for the stop bug: terminateProcess used to send SIGALRM,
// which SIGALRM-ignoring daemons drop. KillPid must actually stop them.
func TestKillPidTerminatesSIGALRMIgnoringDaemon(t *testing.T) {
	cmd, exited := startDaemon(t)

	if err := KillPid(cmd.Process.Pid); err != nil {
		t.Fatalf("KillPid returned error: %v", err)
	}

	select {
	case <-exited:
	case <-time.After(8 * time.Second):
		t.Fatalf("daemon still running after KillPid (pid %d) — stop bug not fixed", cmd.Process.Pid)
	}
}

// Confirms the original behavior was broken: a bare SIGALRM does NOT stop the
// daemon, which is why every service except (Homebrew-managed) PostgreSQL was
// impossible to stop.
func TestSIGALRMDoesNotStopDaemon(t *testing.T) {
	cmd, exited := startDaemon(t)

	cmd.Process.Signal(syscall.SIGALRM) // the old terminateProcess behavior

	select {
	case <-exited:
		t.Fatalf("daemon unexpectedly died on SIGALRM")
	case <-time.After(1500 * time.Millisecond):
		// expected: survived SIGALRM
	}
	cmd.Process.Signal(syscall.SIGKILL)
}

// ReadPidFile must parse the first line of multi-line pid files such as
// PostgreSQL's postmaster.pid.
func TestReadPidFileMultiLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "postmaster.pid")
	content := "12345\n/var/lib/pgsql/data\n1700000000\n5432\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}
	pid, err := ReadPidFile(path)
	if err != nil {
		t.Fatalf("ReadPidFile: %v", err)
	}
	if pid != 12345 {
		t.Fatalf("got pid %d, want 12345", pid)
	}
}
