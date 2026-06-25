package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adtzslowy/luna/internal/config"
)

type Progress func(downloaded, total int64)

func InstallAll(onProgress Progress) error {
	if err := InstallCaddy(onProgress); err != nil {
		return fmt.Errorf("caddy: %w", err)
	}
	if err := InstallPostgreSQL(onProgress); err != nil {
		return fmt.Errorf("postgresql: %w", err)
	}
	if err := InstallMySQL(onProgress); err != nil {
		return fmt.Errorf("mysql: %w", err)
	}
	if err := InstallPHP(onProgress); err != nil {
		return fmt.Errorf("php: %w", err)
	}
	if err := InstallPHPCLI(onProgress); err != nil {
		return fmt.Errorf("php-cli: %w", err)
	}
	if err := InstallAdminer(); err != nil {
		return fmt.Errorf("adminer: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────
// Caddy
// ─────────────────────────────────────────────

func InstallCaddy(onProgress Progress) error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "caddy")
	binPath := filepath.Join(destDir, config.BinExe("caddy"))

	if fileExists(binPath) {
		return fmt.Errorf("caddy already installed at %s", binPath)
	}

	fmt.Println("  → Checking latest Caddy version...")
	version, url, err := FetchCaddyLatest()
	if err != nil {
		return fmt.Errorf("failed to fetch latest caddy: %w", err)
	}

	fmt.Printf("  → Downloading Caddy v%s\n", version)
	fmt.Printf("    %s\n", url)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		if err := extractZip(tmpFile, destDir, "caddy.exe"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := extractTarGz(tmpFile, destDir, "caddy"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
		os.Chmod(binPath, 0755)
	}

	os.WriteFile(filepath.Join(destDir, ".version"), []byte(version), 0644)
	return writeCaddyfile(destDir, base)
}

// ─────────────────────────────────────────────
// PostgreSQL
// ─────────────────────────────────────────────

func InstallPostgreSQL(onProgress Progress) error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "postgresql")
	binPath := filepath.Join(destDir, "bin", config.BinExe("postgres"))

	if fileExists(binPath) {
		return fmt.Errorf("postgresql already installed at %s", binPath)
	}

	fmt.Println("  → Checking latest PostgreSQL version...")
	version, url, err := FetchPostgreSQLLatest()
	if err != nil {
		return fmt.Errorf("failed to fetch latest postgresql: %w", err)
	}

	fmt.Printf("  → Downloading PostgreSQL v%s\n", version)
	fmt.Printf("    %s\n", url)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		if err := extractZipStrip(tmpFile, destDir); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := extractTarGzStrip(tmpFile, destDir); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	}

	os.WriteFile(filepath.Join(destDir, ".version"), []byte(version), 0644)
	return initPostgreSQL(destDir)
}

func initPostgreSQL(pgDir string) error {
	if runtime.GOOS == "darwin" {
		major, err := getMacOSMajor()
		if err == nil && major < 13 {
			return installPostgreSQLHomebrew()
		}
	}

	fmt.Println("  → Initializing PostgreSQL data directory...")
	initdb := filepath.Join(pgDir, "bin", config.BinExe("initdb"))
	dataDir := filepath.Join(pgDir, "data")

	if err := runCommand(initdb,
		"-D", dataDir,
		"-U", "postgres",
		"--no-instructions",
	); err != nil {
		return fmt.Errorf("postgresql initialize failed: %w", err)
	}

	fmt.Println("  → PostgreSQL initialized (user: postgres, no password)")
	return nil
}

// ─────────────────────────────────────────────
// MySQL
// ─────────────────────────────────────────────

func InstallMySQL(onProgress Progress) error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "mysql")
	binPath := filepath.Join(destDir, "bin", config.BinExe("mysqld"))

	if fileExists(binPath) {
		return fmt.Errorf("mysql already installed at %s", binPath)
	}

	// macOS → pakai Homebrew
	if runtime.GOOS == "darwin" {
		return installMySQLHomebrew()
	}

	url, err := MySQLDownloadUrl()
	if err != nil {
		return err
	}

	fmt.Printf("  → Downloading MySQL %s\n", config.MySQLVersion)
	fmt.Printf("    %s\n", url)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		if err := extractZipStrip(tmpFile, destDir); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := extractTarXzStrip(tmpFile, destDir); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	}

	if err := writeMyConf(destDir); err != nil {
		return err
	}

	return initMySQL(destDir)
}

func installMySQLHomebrew() error {
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf(
			"macOS detected — install MySQL via Homebrew:\n" +
				"  brew install mysql",
		)
	}

	fmt.Println("  → macOS detected — installing MySQL via Homebrew...")
	cmd := exec.Command(brewPath, "install", "mysql")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // ignore exit code — brew kadang exit 1 meski sukses

	candidates := []string{
		"/usr/local/opt/mysql/bin",
		"/opt/homebrew/opt/mysql/bin",
	}
	mysqlPath := ""
	for _, p := range candidates {
		if _, err := os.Stat(filepath.Join(p, "mysqld")); err == nil {
			mysqlPath = p
			break
		}
	}
	if mysqlPath == "" {
		return fmt.Errorf("mysql installed but binary not found\nTry: brew link mysql")
	}

	updateMySQLBrewPath(mysqlPath)
	fmt.Printf("  → MySQL found at %s\n", mysqlPath)

	dataDir := filepath.Join(config.BaseDir(), "mysql", "data")
	os.MkdirAll(dataDir, 0755)

	initCmd := exec.Command(filepath.Join(mysqlPath, "mysqld"),
		"--initialize-insecure",
		"--datadir="+dataDir,
	)
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		fmt.Println("  → Data directory already exists, skipping init")
	}

	fmt.Println("  → MySQL ready (root user, no password)")
	return nil
}

func updateMySQLBrewPath(binDir string) {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if svc, ok := cfg.Services["mysql"]; ok {
		svc.BinaryPath = filepath.Join(binDir, "mysqld")
		cfg.Save()
	}
}

func initMySQL(mysqlDir string) error {
	fmt.Println("  → Initializing MySQL data directory...")
	mysqld := filepath.Join(mysqlDir, "bin", config.BinExe("mysqld"))
	dataDir := filepath.Join(mysqlDir, "data")

	if err := runCommand(mysqld,
		"--initialize-insecure",
		"--datadir="+dataDir,
		"--basedir="+mysqlDir,
	); err != nil {
		return fmt.Errorf("mysql initialize failed: %w", err)
	}

	fmt.Println("  → MySQL initialized (root user, no password)")
	return nil
}

// ─────────────────────────────────────────────
// PHP-FPM
// ─────────────────────────────────────────────

func InstallPHP(onProgress Progress) error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "php")
	binPath := filepath.Join(destDir, config.BinExe("php-fpm"))

	if fileExists(binPath) {
		return fmt.Errorf("php-fpm already installed at %s", binPath)
	}

	fmt.Println("  → Fetching latest PHP-FPM...")
	_, url, err := FetchPHPLatest()
	if err != nil {
		return fmt.Errorf("failed to get PHP URL: %w", err)
	}

	fmt.Printf("  → Downloading PHP-FPM\n    %s\n", url)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		if err := extractZip(tmpFile, destDir, "php-fpm.exe"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := extractTarGz(tmpFile, destDir, "php-fpm"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
		os.Chmod(binPath, 0755)
	}

	return writePHPFpmConf(destDir, base)
}

func writePHPFpmConf(phpDir, baseDir string) error {
	conf := fmt.Sprintf(`[global]
pid = %s
error_log = %s
daemonize = no

[www]
listen = 127.0.0.1:9000
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
`,
		filepath.Join(phpDir, "php-fpm.pid"),
		filepath.Join(phpDir, "php-fpm.log"),
	)
	return os.WriteFile(filepath.Join(phpDir, "php-fpm.conf"), []byte(conf), 0644)
}

// ─────────────────────────────────────────────
// PHP CLI
// ─────────────────────────────────────────────

func InstallPHPCLI(onProgress Progress) error {
	base := config.BaseDir()
	binDir := filepath.Join(base, "bin")
	binPath := filepath.Join(binDir, config.BinExe("php"))

	if fileExists(binPath) {
		return fmt.Errorf("php CLI already installed at %s", binPath)
	}

	fmt.Println("  → Fetching latest PHP CLI...")
	_, url, err := FetchPHPCLILatest()
	if err != nil {
		return fmt.Errorf("failed to get PHP CLI URL: %w", err)
	}

	fmt.Printf("  → Downloading PHP CLI\n    %s\n", url)

	tmpFile, err := downloadFile(url, onProgress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		if err := extractZip(tmpFile, binDir, "php.exe"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
	} else {
		if err := extractTarGz(tmpFile, binDir, "php"); err != nil {
			return fmt.Errorf("extract failed: %w", err)
		}
		os.Chmod(binPath, 0755)
	}

	fmt.Printf("  → PHP CLI installed at %s\n", binPath)
	return nil
}

// ─────────────────────────────────────────────
// Adminer
// ─────────────────────────────────────────────

func InstallAdminer() error {
	base := config.BaseDir()
	adminerDir := filepath.Join(base, "www", "adminer")
	adminerFile := filepath.Join(adminerDir, "index.php")

	os.Remove(adminerFile) // hapus versi lama kalau ada

	if err := os.MkdirAll(adminerDir, 0755); err != nil {
		return err
	}

	fmt.Println("  → Downloading Adminer v5.4.2...")
	resp, err := http.Get("https://github.com/vrana/adminer/releases/download/v5.4.2/adminer-5.4.2.php")
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(adminerFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("  → Adminer installed at http://localhost/adminer/")
	return nil
}

// ─────────────────────────────────────────────
// LunaBase
// ─────────────────────────────────────────────

func InstallLunaBase() error {
	base := config.BaseDir()
	destDir := filepath.Join(base, "www", "lunabase")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	fmt.Println("  → Installing LunaBase...")
	fmt.Printf("  → Copy index.php ke %s\n", destDir)
	fmt.Println("  → LunaBase siap di http://localhost/lunabase/")
	return nil
}

// ─────────────────────────────────────────────
// Download
// ─────────────────────────────────────────────

func downloadFile(url string, onProgress Progress) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "luna-download-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	var downloaded int64
	total := resp.ContentLength
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				return "", werr
			}
			downloaded += int64(n)
			if onProgress != nil {
				onProgress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return tmp.Name(), nil
}

// ─────────────────────────────────────────────
// Extract
// ─────────────────────────────────────────────

func extractTarGz(src, destDir, targetFile string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == targetFile {
			out, err := os.Create(filepath.Join(destDir, targetFile))
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			return err
		}
	}
	return fmt.Errorf("file %s not found in archive", targetFile)
}

func extractTarGzStrip(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		target := filepath.Join(destDir, parts[1])
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
			os.Chmod(target, os.FileMode(hdr.Mode))
		}
	}
	return nil
}

func extractTarXzStrip(src, destDir string) error {
	cmd := exec.Command("tar", "-xJf", src, "-C", destDir, "--strip-components=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func extractZip(src, destDir, targetFile string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == targetFile {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(filepath.Join(destDir, targetFile))
			if err != nil {
				rc.Close()
				return err
			}
			io.Copy(out, rc)
			out.Close()
			rc.Close()
			return nil
		}
	}
	return fmt.Errorf("file %s not found in zip", targetFile)
}

func extractZipStrip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		target := filepath.Join(destDir, parts[1])
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

// ─────────────────────────────────────────────
// Config generators
// ─────────────────────────────────────────────

func writeCaddyfile(caddyDir, baseDir string) error {
	wwwDir := filepath.Join(baseDir, "www")
	os.MkdirAll(wwwDir, 0755)

	indexPath := filepath.Join(wwwDir, "index.html")
	if !fileExists(indexPath) {
		os.WriteFile(indexPath, []byte(`<!DOCTYPE html>
<html>
<head><title>Luna</title></head>
<body><h1>🌙 Luna is running!</h1><p>Place your files in ~/.luna/www/</p></body>
</html>`), 0644)
	}

	caddyfile := fmt.Sprintf(`# Luna - Caddyfile

:80 {
	root * %s
	file_server
	php_fastcgi localhost:9000
	log {
		output file %s
	}
}
`, wwwDir, filepath.Join(caddyDir, "caddy.log"))

	return os.WriteFile(filepath.Join(caddyDir, "Caddyfile"), []byte(caddyfile), 0644)
}

func writeMyConf(mysqlDir string) error {
	dataDir := filepath.Join(mysqlDir, "data")
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(filepath.Join(mysqlDir, "logs"), 0755)

	conf := fmt.Sprintf(`[mysqld]
basedir=%s
datadir=%s
port=3306
pid-file=%s
log-error=%s
socket=%s

innodb_buffer_pool_size=128M
max_connections=50
`,
		mysqlDir,
		dataDir,
		filepath.Join(dataDir, "mysql.pid"),
		filepath.Join(dataDir, "mysql-error.log"),
		filepath.Join(mysqlDir, "mysql.sock"),
	)

	return os.WriteFile(filepath.Join(mysqlDir, "my.cnf"), []byte(conf), 0644)
}

// ─────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
