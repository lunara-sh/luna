package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adtzslowy/luna/internal/config"
)

type UpdateResult struct {
	Service   string
	Installed string
	Latest    string
	Updated   bool
	Error     error
}

func CheckUpdates() ([]UpdateResult, error) {
	var results []UpdateResult

	// Check Caddy
	installed := readVersion(filepath.Join(config.BaseDir(), "caddy", ".version"))
	latest, _, err := FetchCaddyLatest()
	results = append(results, UpdateResult{
		Service:   "caddy",
		Installed: installed,
		Latest:    latest,
		Error:     err,
	})

	return results, nil
}

func UpdateCaddy(onProgress Progress) error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "caddy")
	binPath := filepath.Join(destDir, config.BinExe("caddy"))

	fmt.Println("  → Checking latest Caddy version...")
	version, url, err := FetchCaddyLatest()
	if err != nil {
		return err
	}

	installed := readVersion(filepath.Join(destDir, ".version"))
	if installed == version {
		fmt.Printf("  → Caddy v%s is already up to date\n", version)
		return nil
	}

	fmt.Printf("  → Updating Caddy %s → %s\n", installed, version)

	// Backup binary lama
	backupPath := binPath + ".bak"
	os.Rename(binPath, backupPath)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		os.Rename(backupPath, binPath) // restore backup
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if isWindows() {
		if err := extractZip(tmpFile, destDir, "caddy.exe"); err != nil {
			os.Rename(backupPath, binPath)
			return err
		}
	} else {
		if err := extractTarGz(tmpFile, destDir, "caddy"); err != nil {
			os.Rename(backupPath, binPath)
			return err
		}
		os.Chmod(binPath, 0o755)
	}

	os.Remove(backupPath)
	os.WriteFile(filepath.Join(destDir, ".version"), []byte(version), 0o644)

	fmt.Printf("  → Caddy updated to v%s\n", version)
	return nil
}

func readVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

func isWindows() bool {
	return filepath.Separator == '\\'
}
