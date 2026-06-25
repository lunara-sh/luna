package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/adtzslowy/luna/internal/config"
	"github.com/adtzslowy/luna/internal/process"
)

type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateUnknown State = "unknown"
)

type ServiceStatus struct {
	Name     string
	Display  string
	State    State
	PID      int
	Port     int
	PortOpen bool
}

type Manager struct {
	cfg  *config.Config
	cmds map[string]*exec.Cmd
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:  cfg,
		cmds: make(map[string]*exec.Cmd),
	}
}

func (m *Manager) Start(name string) error {
	svc, ok := m.cfg.Services[name]
	if !ok {
		return fmt.Errorf("unknown service: %s", name)
	}

	status := m.Status(name)
	if status.State == StateRunning {
		return fmt.Errorf("%s is already running (PID %d)", svc.Name, status.PID)
	}

	if _, err := os.Stat(svc.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s\nRun 'luna install %s' to download it", svc.BinaryPath, name)
	}

	if svc.LogFile != "" {
		os.MkdirAll(filepath.Dir(svc.LogFile), 0o755)
	}

	cmd := exec.Command(svc.BinaryPath, svc.Args...)
	cmd.Dir = filepath.Dir(svc.BinaryPath)

	if svc.LogFile != "" {
		lf, err := os.OpenFile(svc.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			cmd.Stdout = lf
			cmd.Stderr = lf
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", svc.Name, err)
	}

	m.cmds[name] = cmd

	if svc.PidFile != "" {
		os.MkdirAll(filepath.Dir(svc.PidFile), 0o755)
		os.WriteFile(svc.PidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644)
	}

	time.Sleep(500 * time.Millisecond)
	if !process.IsProcessRunning(cmd.Process.Pid) {
		return fmt.Errorf("%s started but exited immediately — check log: %s", svc.Name, svc.LogFile)
	}

	return nil
}

func (m *Manager) Restart(name string) error {
	if _, ok := m.cfg.Services[name]; !ok {
		return fmt.Errorf("unknown service: %s", name)
	}

	status := m.Status(name)
	if status.State == StateRunning {
		if err := m.Stop(name); err != nil {
			return fmt.Errorf("restart: stop failed: %w", err)
		}
		time.Sleep(800 * time.Millisecond)
	}

	return m.Start(name)
}

func (m *Manager) Status(name string) ServiceStatus {
	svc, ok := m.cfg.Services[name]
	if !ok {
		return ServiceStatus{Name: name, State: StateUnknown}
	}

	status := ServiceStatus{
		Name:    name,
		Display: svc.Name,
		Port:    svc.Port,
	}

	pid := 0
	if svc.PidFile != "" {
		if p, err := process.ReadPidFile(svc.PidFile); err == nil {
			pid = p
		}
	}

	if pid == 0 {
		if cmd, ok := m.cmds[name]; ok && cmd.Process != nil {
			pid = cmd.Process.Pid
		}
	}

	if pid > 0 && process.IsProcessRunning(pid) {
		status.State = StateRunning
		status.PID = pid
	} else {
		status.State = StateStopped
	}

	status.PortOpen = process.IsPortOpen(svc.Port)
	if status.State == StateStopped && status.PortOpen {
		status.State = StateRunning
	}

	return status
}

func (m *Manager) Stop(name string) error {
	svc, ok := m.cfg.Services[name]
	if !ok {
		return fmt.Errorf("unknown service: %s", name)
	}

	status := m.Status(name)
	if status.State == StateStopped {
		return fmt.Errorf("%s is not running", svc.Name)
	}

	// Deteksi Homebrew binary dan stop via brew services
	if isHomebrewBinary(svc.BinaryPath) {
		return stopViaHomebrew(name, svc.BinaryPath)
	}

	if status.PID > 0 {
		if err := process.KillPid(status.PID); err != nil {
			return fmt.Errorf("failed to stop %s: %w", svc.Name, err)
		}
	} else {
		binName := filepath.Base(svc.BinaryPath)
		pids, err := process.FindProcessByName(binName)
		if err != nil || len(pids) == 0 {
			return fmt.Errorf("cannot find %s process to stop", svc.Name)
		}
		for _, pid := range pids {
			process.KillPid(pid)
		}
	}

	if svc.PidFile != "" {
		os.Remove(svc.PidFile)
	}

	delete(m.cmds, name)
	return nil
}

func isHomebrewBinary(binaryPath string) bool {
	return strings.Contains(binaryPath, "/opt/homebrew/") ||
		strings.Contains(binaryPath, "/usr/local/opt/")
}

func stopViaHomebrew(name, binaryPath string) error {
	// Extract service name dari path
	// e.g. /usr/local/opt/postgresql@16/bin/postgres → postgresql@16
	brewSvc := ""
	if strings.Contains(binaryPath, "postgresql") {
		brewSvc = "postgresql@16"
	} else if strings.Contains(binaryPath, "mysql") {
		brewSvc = "mysql"
	}

	if brewSvc == "" {
		return fmt.Errorf("cannot determine homebrew service name for %s", name)
	}

	cmd := exec.Command("brew", "services", "stop", brewSvc)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) StatusAll() []ServiceStatus {
	order := []string{"caddy", "php", "postgresql", "mysql"}
	var statuses []ServiceStatus

	shown := map[string]bool{}
	for _, name := range order {
		if _, ok := m.cfg.Services[name]; ok {
			statuses = append(statuses, m.Status(name))
			shown[name] = true
		}
	}

	for name := range m.cfg.Services {
		if !shown[name] {
			statuses = append(statuses, m.Status(name))
		}
	}

	return statuses
}

func (m *Manager) StartAll() []error {
	order := []string{"postgresql", "mysql", "php", "caddy"}
	var errs []error
	started := map[string]bool{}

	for _, name := range order {
		if _, ok := m.cfg.Services[name]; ok {
			if err := m.Start(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
			started[name] = true
		}
	}

	for name := range m.cfg.Services {
		if !started[name] {
			if err := m.Start(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	return errs
}

func (m *Manager) StopAll() []error {
	order := []string{"caddy", "php", "postgresql", "mysql"}
	var errs []error
	stopped := map[string]bool{}

	for _, name := range order {
		if _, ok := m.cfg.Services[name]; ok {
			if err := m.Stop(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
			stopped[name] = true
		}
	}

	for name := range m.cfg.Services {
		if !stopped[name] {
			if err := m.Stop(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	return errs
}
