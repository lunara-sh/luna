package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adtzslowy/luna/internal/browser"
	"github.com/adtzslowy/luna/internal/config"
	"github.com/adtzslowy/luna/internal/install"
	"github.com/adtzslowy/luna/internal/logs"
	"github.com/adtzslowy/luna/internal/service"
	"github.com/adtzslowy/luna/internal/ui"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		cmdStatus(nil)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load config: " + err.Error())
		os.Exit(1)
	}

	mgr := service.NewManager(cfg)

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "start":
		cmdStart(mgr, args)
	case "stop":
		cmdStop(mgr, args)
	case "restart":
		cmdRestart(mgr, args)
	case "status":
		cmdStatus(mgr)
	case "config":
		cmdConfig(cfg, args)
	case "open":
		cmdOpen(args)
	case "logs":
		cmdLogs(mgr, args)
	case "install":
		cmdInstall(args)
	case "update":
		cmdUpdate(args)
	case "check":
		fmt.Print("  → Checking latest Caddy version... ")
		version, _, err := install.FetchCaddyLatest()
		if err != nil {
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess(fmt.Sprintf("Latest Caddy: v%s", version))

		versionFile := filepath.Join(config.BaseDir(), "caddy", ".version")
		if data, err := os.ReadFile(versionFile); err == nil {
			installed := strings.TrimSpace(string(data))
			if installed == version {
				ui.PrintInfo("Already up to date!")
			} else {
				ui.PrintWarn(fmt.Sprintf("Installed: v%s → Update available!", installed))
				ui.PrintInfo("Run 'luna install caddy' to update")
			}
		}
	case "version", "-v", "--version":
		fmt.Printf("Luna v%s\n", version)
	case "help", "-h", "--help":
		printHelp()
	default:
		ui.PrintError(fmt.Sprintf("Unknown command: %s", cmd))
		fmt.Println()
		printHelp()
		os.Exit(1)
	}
}

func cmdStart(mgr *service.Manager, args []string) {
	if len(args) == 0 || args[0] == "all" {
		ui.PrintHeader("Starting all services")
		errs := mgr.StartAll()
		printServiceResults(mgr, errs)
		return
	}

	name := strings.ToLower(args[0])
	spin := ui.NewSpinner(fmt.Sprintf("Starting %s...", name))
	spin.Start()

	if err := mgr.Start(name); err != nil {
		spin.Stop(false, err.Error())
		os.Exit(1)
	}

	st := mgr.Status(name)
	spin.Stop(true, fmt.Sprintf("%s started (PID %d)", st.Display, st.PID))
}

func cmdStop(mgr *service.Manager, args []string) {
	if len(args) == 0 || args[0] == "all" {
		ui.PrintHeader("Stopping all services")
		errs := mgr.StopAll()
		printServiceResults(mgr, errs)
		return
	}

	name := strings.ToLower(args[0])
	spin := ui.NewSpinner(fmt.Sprintf("Stopping %s...", name))
	spin.Start()

	if err := mgr.Stop(name); err != nil {
		spin.Stop(false, err.Error())
		os.Exit(1)
	}

	spin.Stop(true, fmt.Sprintf("%s stopped", name))
}

func cmdRestart(mgr *service.Manager, args []string) {
	if len(args) == 0 || args[0] == "all" {
		ui.PrintHeader("Restarting all services")
		mgr.StopAll()
		errs := mgr.StartAll()
		printServiceResults(mgr, errs)
		return
	}

	name := strings.ToLower(args[0])
	spin := ui.NewSpinner(fmt.Sprintf("Restarting %s...", name))
	spin.Start()

	if err := mgr.Restart(name); err != nil {
		spin.Stop(false, err.Error())
		os.Exit(1)
	}

	st := mgr.Status(name)
	spin.Stop(true, fmt.Sprintf("%s restarted (PID %d)", st.Display, st.PID))
}

func cmdStatus(mgr *service.Manager) {
	ui.PrintBanner()

	if mgr == nil {
		cfg, err := config.Load()
		if err != nil {
			ui.PrintError("Failed to load config: " + err.Error())
			return
		}
		mgr = service.NewManager(cfg)
	}

	ui.PrintHeader("Service Status")

	statuses := mgr.StatusAll()
	for _, s := range statuses {
		running := s.State == service.StateRunning
		ui.PrintStatus(s.Name, s.Display, running, s.PID, s.Port, s.PortOpen)
	}

	fmt.Println()
	ui.PrintFooter()
}

func cmdConfig(cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	showFlag := fs.Bool("show", false, "Show current config")
	initFlag := fs.Bool("init", false, "Write default config to disk")
	fs.Parse(args)

	if *initFlag {
		if err := cfg.Save(); err != nil {
			ui.PrintError("Failed to write config: " + err.Error())
			os.Exit(1)
		}
		ui.PrintSuccess("Config written to " + config.ConfigPath())
		return
	}

	if *showFlag || len(args) == 0 {
		fmt.Printf("Config file : %s\n\n", config.ConfigPath())
		fmt.Printf("Base dir    : %s\n\n", cfg.BaseDir)
		for name, svc := range cfg.Services {
			fmt.Printf("[%s]\n", name)
			fmt.Printf("  Binary : %s\n", svc.BinaryPath)
			fmt.Printf("  Config : %s\n", svc.ConfigPath)
			fmt.Printf("  Port   : %d\n", svc.Port)
			fmt.Printf("  PID    : %s\n", svc.PidFile)
			fmt.Printf("  Log    : %s\n\n", svc.LogFile)
		}
	}
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func printServiceResults(mgr *service.Manager, errs []error) {
	fmt.Println()
	for _, s := range mgr.StatusAll() {
		running := s.State == service.StateRunning
		ui.PrintStatus(s.Name, s.Display, running, s.PID, s.Port, s.PortOpen)
	}
	if len(errs) > 0 {
		fmt.Println()
		for _, e := range errs {
			ui.PrintWarn(e.Error())
		}
	}
	fmt.Println()
}

func printHelp() {
	ui.PrintBanner()
	fmt.Println(ui.Bold_("USAGE"))
	fmt.Println("  luna <command> [service] [flags]")
	fmt.Println()
	fmt.Println(ui.Bold_("COMMANDS"))
	fmt.Printf("  %-30s %s\n", "start [service|all]", "Start a service or all services")
	fmt.Printf("  %-30s %s\n", "stop [service|all]", "Stop a service or all services")
	fmt.Printf("  %-30s %s\n", "restart [service|all]", "Restart a service or all services")
	fmt.Printf("  %-30s %s\n", "status", "Show status of all services")
	fmt.Printf("  %-30s %s\n", "config [--show|--init]", "Show or write config")
	fmt.Printf("  %-30s %s\n", "version", "Show version")
	fmt.Printf("	%-30s %s\n", "open [lunabase|adminer]", "Buka browser ke service")
	fmt.Println()
	fmt.Println(ui.Bold_("SERVICES"))
	fmt.Printf("  %-12s Caddy web server    (port 80)\n", "caddy")
	fmt.Printf("  %-12s PostgreSQL          (port 5432)\n", "postgresql")
	fmt.Printf("  %-12s PHP-FPM             (port 9000)\n", "php")
	fmt.Printf("  %-12s PHP             (port 9000)\n", "php")
	fmt.Println()
	fmt.Println(ui.Bold_("EXAMPLES"))
	fmt.Println("  luna install          # install semua (caddy + postgresql + php + adminer)")
	fmt.Println("  luna install caddy    # install hanya Caddy")
	fmt.Println("  luna install mysql    # install PostgreSQL")
	fmt.Println("  luna install php      # install PHP-FPM")
	fmt.Println("  luna install php-cli      # install PHP CLI")
	fmt.Println("  luna install adminer  # install Adminer")
	fmt.Println()
	ui.PrintFooter()
}

func cmdInstall(args []string) {
	ui.PrintBanner()

	// Progress bar sederhana
	onProgress := func(downloaded, total int64) {
		if total <= 0 {
			fmt.Printf("\r  → Downloaded %s", formatBytes(downloaded))
			return
		}
		pct := float64(downloaded) / float64(total) * 100
		filled := int(pct / 5) // 20 chars wide
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
		fmt.Printf(
			"\r  [%s] %.1f%% (%s / %s)",
			bar, pct,
			formatBytes(downloaded),
			formatBytes(total),
		)
	}

	if len(args) == 0 || args[0] == "all" {
		ui.PrintHeader("Installing all services")
		if err := install.InstallAll(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("All services installed!")
		ui.PrintInfo("Run 'luna start' to start everything")
		return
	}

	name := strings.ToLower(args[0])
	switch name {
	case "caddy":
		ui.PrintHeader("Installing Caddy")
		if err := install.InstallCaddy(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("Caddy installed!")
	case "mysql":
		ui.PrintHeader("Installing MySQL")
		if err := install.InstallPostgreSQL(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("MySQL installed!")
	case "php":
		ui.PrintHeader("Installing PHP-FPM")
		if err := install.InstallPHP(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("PHP-FPM installed!")

	case "php-cli":
		ui.PrintHeader("Installing PHP-CLI")
		if err := install.InstallPHPCLI(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("PHP CLI installed!")

	case "adminer":
		ui.PrintHeader("Installing Adminer")
		if err := install.InstallAdminer(); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("Adminer installed!")
		ui.PrintInfo("Access at http://localhost/adminer/")
	default:
		ui.PrintError(fmt.Sprintf("Unknown service: %s", name))
		ui.PrintInfo("Available: caddy, mysql, all")
		os.Exit(1)
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func cmdOpen(args []string) {
	target := "http://localhost"

	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "lunabase", "db":
			target = "http://localhost/lunabase"
		case "adminer":
			target = "http://localhost/adminer"
		default:
			target = "http://localhost/" + args[0]
		}
	}

	ui.PrintInfo(fmt.Sprintf("Opening %s...", target))
	if err := openBrowser(target); err != nil {
		ui.PrintError("Failed to open the browser: " + err.Error())
		os.Exit(1)
	}
}

func openBrowser(url string) error {
	return browser.Open(url)
}

func cmdLogs(mgr *service.Manager, args []string) {
	follow := false
	lines := 50
	name := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f", "--follow":
			follow = true
		case "-n", "--lines":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					lines = n
				}
				i++
			}
		default:
			name = args[i]
		}
	}

	if name == "" {
		ui.PrintError("Usage: luna logs <service> [-f] [-n lines]")
		ui.PrintInfo("Services: caddy, postgresql, mysql, php")
		os.Exit(1)
	}

	cfg, _ := config.Load()
	svc, ok := cfg.Services[name]
	if !ok {
		ui.PrintError(fmt.Sprintf("Unknown service: %s", name))
		os.Exit(1)
	}

	if svc.LogFile == "" {
		ui.PrintError(fmt.Sprintf("No log file configured for %s", name))
		os.Exit(1)
	}

	ui.PrintInfo(fmt.Sprintf("Logs for %s (%s)", svc.Name, svc.LogFile))
	fmt.Println()

	if err := logs.Tail(svc.LogFile, follow, lines); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
}

func cmdUpdate(args []string) {
	ui.PrintBanner()

	onProgress := func(downloaded, total int64) {
		if total <= 0 {
			fmt.Printf("\r  → Downloaded %s", formatBytes(downloaded))
			return
		}
		pct := float64(downloaded) / float64(total) * 100
		filled := int(pct / 5)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
		fmt.Printf("\r  [%s] %.1f%% (%s / %s)", bar, pct, formatBytes(downloaded), formatBytes(total))
	}

	if len(args) == 0 || args[0] == "all" {
		ui.PrintHeader("Checking for updates")

		results, _ := install.CheckUpdates()
		fmt.Println()
		for _, r := range results {
			if r.Error != nil {
				ui.PrintWarn(fmt.Sprintf("%-12s failed to check: %s", r.Service, r.Error))
				continue
			}
			if r.Installed == r.Latest {
				ui.PrintSuccess(fmt.Sprintf("%-12s v%s (up to date)", r.Service, r.Installed))
			} else {
				ui.PrintWarn(fmt.Sprintf("%-12s v%s → v%s (update available)", r.Service, r.Installed, r.Latest))
			}
		}

		fmt.Println()
		ui.PrintInfo("Run 'luna update caddy' to update")
		return
	}

	name := strings.ToLower(args[0])
	switch name {
	case "caddy":
		ui.PrintHeader("Updating Caddy")
		if err := install.UpdateCaddy(onProgress); err != nil {
			fmt.Println()
			ui.PrintError(err.Error())
			os.Exit(1)
		}
		fmt.Println()
		ui.PrintSuccess("Caddy updated!")
	default:
		ui.PrintError(fmt.Sprintf("Update not supported for: %s", name))
		ui.PrintInfo("Currently supported: caddy")
		os.Exit(1)
	}
}
