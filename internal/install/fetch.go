package install

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

const githubCaddyAPI = "https://api.github.com/repos/caddyserver/caddy/releases/latest"
const postgresAPI = "https://api.github.com/repos/theseus-rs/postgresql-binaries/releases/latest"
const staticPHPBaseUrl = "https://dl.static-php.dev/static-php-cli/bulk/"
const githubMySQLApi = "https://api.github.com/repos/theseus-rs/mysql-binaries/releases/latest"

const MySQLVersion = "8.4.5"

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type postgresRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func FetchCaddyLatest() (string, string, error) {
	req, err := http.NewRequest("GET", githubCaddyAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "luna-dev-manager")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to parse Github response: %w", err)
	}
	version := strings.TrimPrefix(release.TagName, "v")

	fmt.Println("   [debug] available assets:")
	for _, a := range release.Assets {
		fmt.Printf("     - %s\n", a.Name)
	}

	url, err := findCaddyAsset(release.Assets, version)
	if err != nil {
		return "", "", err
	}

	return version, url, nil
}

func findCaddyAsset(assets []githubAsset, version string) (string, error) {
	os_ := runtime.GOOS
	arch := runtime.GOARCH

	osMap := map[string]string{
		"darwin":  "mac",
		"linux":   "linux",
		"windows": "windows",
	}

	o, ok := osMap[os_]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", os_)
	}

	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
		"arm":   "armv6",
	}
	a, ok := archMap[arch]
	if !ok {
		return "", fmt.Errorf("unsupported arch: %s", arch)
	}

	ext := "tar.gz"
	if os_ == "windows" {
		ext = "zip"
	}

	target := fmt.Sprintf("caddy_%s_%s_%s.%s", version, o, a, ext)

	for _, asset := range assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	for _, asset := range assets {
		name := asset.Name
		if strings.Contains(name, os_) &&
			strings.Contains(name, a) &&
			strings.HasSuffix(name, ext) &&
			!strings.Contains(name, ".sha") &&
			!strings.Contains(name, ".sig") {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf(
		"no asset found for %s/%s\nlooking for: %s\ncheck https://github.com/caddyserver/caddy/releases/latest",
		os_, arch, target,
	)
}

func FetchPostgreSQLLatest() (string, string, error) {
	req, _ := http.NewRequest("GET", postgresAPI, nil)
	req.Header.Set("User-Agent", "luna-dev-manager")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	var release postgresRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")
	url, err := findPostgresAsset(release.Assets, version)
	if err != nil {
		return "", "", err
	}

	return version, url, nil
}

func findPostgresAsset(assets []githubAsset, version string) (string, error) {
	for _, a := range assets {
		if strings.Contains(a.Name, "darwin") || strings.Contains(a.Name, "apple") {
			fmt.Printf("  [debug] %s\n", a.Name)
		}
	}

	tripleMap := map[string]map[string]string{
		"darwin": {
			"amd64": "x86_64-apple-darwin",
			"arm64": "aarch64-apple-darwin",
		},
		"linux": {
			"amd64": "x86_64-unknown-linux-gnu",
			"arm64": "aarch64-unknown-linux-gnu",
		},
		"windows": {
			"amd64": "x86_64-pc-windows-msvc",
			"arm64": "aarch64-pc-windows-msvc",
		},
	}

	os_ := runtime.GOOS
	arch := runtime.GOARCH

	osTriples, ok := tripleMap[os_]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", os_)
	}
	triple, ok := osTriples[arch]
	if !ok {
		return "", fmt.Errorf("unsupported arch: %s", arch)
	}

	ext := "tar.gz"
	if os_ == "windows" {
		ext = "zip"
	}

	target := fmt.Sprintf("postgresql-%s-%s.%s", version, triple, ext)

	for _, asset := range assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no PostgreSQL binary found for %s/%s\nlooking for: %s", os_, arch, target)
}

func FetchPHPLatest() (string, string, error) {
	// static-php.dev tidak punya API, kita fetch index halaman
	// dan cari versi terbaru php-fpm untuk platform ini
	target, err := buildPHPURL()
	if err != nil {
		return "", "", err
	}
	return "latest", target, nil
}

func buildPHPURL() (string, error) {
	os_ := runtime.GOOS
	arch := runtime.GOARCH

	// OS name map
	osMap := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}
	osName, ok := osMap[os_]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", os_)
	}

	// Arch map
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}
	archName, ok := archMap[arch]
	if !ok {
		return "", fmt.Errorf("unsupported arch: %s", arch)
	}

	resp, err := http.Get(staticPHPBaseUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PHP index: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	ext := "tar.gz"
	if os_ == "windows" {
		ext = "zip"
		archName = "x64"
	}

	suffix := fmt.Sprintf("-fpm-%s-%s.%s", osName, archName, ext)

	latest := ""
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, suffix) && strings.Contains(line, "php-8.") {
			start := strings.Index(line, "php-8.")
			if start < 0 {
				continue
			}
			end := strings.Index(line[start:], suffix)
			if end < 0 {
				continue
			}
			filename := line[start : start+end+len(suffix)]
			if filename > latest {
				latest = filename
			}
		}
	}

	if latest == "" {
		// Fallback ke URL yang kita construct manual dengan versi known
		latest = fmt.Sprintf("php-8.4.0-fpm-%s-%s.%s", osName, archName, ext)
	}

	return staticPHPBaseUrl + latest, nil
}

func FetchPHPCLILatest() (string, string, error) {
	url, err := buildPHPCLIURL()
	if err != nil {
		return "", "", err
	}
	return "latest", url, nil
}

func buildPHPCLIURL() (string, error) {
	os_ := runtime.GOOS
	arch := runtime.GOARCH

	osMap := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}
	osName, ok := osMap[os_]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", os_)
	}

	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}
	archName, ok := archMap[arch]
	if !ok {
		return "", fmt.Errorf("unsupported arch: %s", arch)
	}

	ext := "tar.gz"
	if os_ == "windows" {
		ext = "zip"
		archName = "x64"
	}

	// Fetch index untuk cari versi terbaru
	resp, err := http.Get(staticPHPBaseUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PHP index: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	suffix := fmt.Sprintf("-cli-%s-%s.%s", osName, archName, ext)

	latest := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, suffix) && strings.Contains(line, "php-8.") {
			start := strings.Index(line, "php-8.")
			if start < 0 {
				continue
			}
			end := strings.Index(line[start:], suffix)
			if end < 0 {
				continue
			}
			filename := line[start : start+end+len(suffix)]
			if filename > latest {
				latest = filename
			}
		}
	}

	if latest == "" {
		latest = fmt.Sprintf("php-8.4.0-cli-%s-%s.%s", osName, archName, ext)
	}

	return staticPHPBaseUrl + latest, nil
}

func MySQLDownloadUrl() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "", fmt.Errorf("Homebrew")
	case "windows":
		return fmt.Sprintf(
			"https://dev.mysql.com/get/Downloads/MySQL-8.4/mysql-%s-winx64.zip",
			MySQLVersion,
		), nil
	default:
		return fmt.Sprintf(
			"https://dev.mysql.com/get/Downloads/MySQL-8.4/mysql-%s-linux-glibc2.28-x86_64.tar.xz",
			MySQLVersion,
		), nil
	}
}
