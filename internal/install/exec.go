package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adtzslowy/luna/internal/config"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkMacOSVersion(minMajor int) error {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return nil // kalau gagal cek, lanjut aja
	}

	version := strings.TrimSpace(string(out))
	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return nil
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}

	if major < minMajor {
		return fmt.Errorf(
			"Luna requires macOS %d or later for bundled PostgreSQL\n"+
				"  Your macOS : %s\n"+
				"  Alternative: brew install postgresql@16",
			minMajor, version,
		)
	}
	return nil
}

func getMacOSMajor() (int, error) {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ".")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid version")
	}
	return strconv.Atoi(parts[0])
}

func installPostgreSQLHomebrew() error {
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf(
			"macOS 12 detected — bundled PostgreSQL requires macOS 13+\n" +
				"  Homebrew not found. Install it from https://brew.sh then run:\n" +
				"  brew install postgresql@16",
		)
	}

	fmt.Println("  → macOS 12 detected — using Homebrew to install PostgreSQL...")
	fmt.Println("  → Running: brew install postgresql@16")

	cmd := exec.Command(brewPath, "install", "postgresql@16")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	pgPath := findHomebrewPostgres()
	if pgPath == "" {
		return fmt.Errorf(
			"postgresql@16 not found after install\n" +
				"  Try manually: brew install postgresql@16\n" +
				"  Then: brew link postgresql@16",
		)
	}

	fmt.Printf("  → PostgreSQL found at %s\n", pgPath)

	dataDir := "/usr/local/var/postgresql@16"
	if _, err := os.Stat(dataDir); err == nil {
		fmt.Println("  → Data directory already initialized by Homebrew")
		updateBrewBinaryPath(pgPath)
		return nil
	}

	fmt.Println("  → Initializing data directory...")
	dataDir = filepath.Join(config.BaseDir(), "mysql", "data")
	os.MkdirAll(dataDir, 0755)

	initdb := filepath.Join(pgPath, "initdb")
	cmd2 := exec.Command(initdb, "-D", dataDir, "-U", "postgres", "--no-instructions")
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("initdb failed: %w", err)
	}

	updateBrewBinaryPath(pgPath)
	fmt.Println("  → PostgreSQL initialized (user: postgres, no password)")
	return nil
}

func findHomebrewPostgres() string {
	// Cek Intel Mac path
	candidates := []string{
		"/usr/local/opt/postgresql@16/bin",
		"/usr/local/opt/postgresql/bin",
		// Apple Silicon path (untuk referensi)
		"/opt/homebrew/opt/postgresql@16/bin",
		"/opt/homebrew/opt/postgresql/bin",
	}
	for _, p := range candidates {
		if _, err := os.Stat(filepath.Join(p, "initdb")); err == nil {
			return p
		}
	}
	return ""
}

func updateBrewBinaryPath(binDir string) {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if svc, ok := cfg.Services["mysql"]; ok {
		svc.BinaryPath = filepath.Join(binDir, "postgres")
		cfg.Save()
	}
}
