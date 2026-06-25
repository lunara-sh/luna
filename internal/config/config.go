package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const (
	CaddyVersion = "2.11.4"
	MySQLVersion = "8.4.5"
)

type ServiceConfig struct {
	Name       string   `json:"name"`
	BinaryPath string   `json:"binary_path"`
	ConfigPath string   `json:"config_path"`
	Port       int      `json:"port"`
	Args       []string `json:"args"`
	PidFile    string   `json:"pid_file"`
	LogFile    string   `json:"log_file"`
}

type Config struct {
	BaseDir  string                    `json:"base_dir"`
	Services map[string]*ServiceConfig `json:"services"`
}

func DefaultConfig() *Config {
	base := BaseDir()

	return &Config{
		BaseDir: base,
		Services: map[string]*ServiceConfig{
			"caddy": {
				Name:       "Caddy",
				BinaryPath: filepath.Join(base, "caddy", binExe("caddy")),
				ConfigPath: filepath.Join(base, "caddy", "Caddyfile"),
				Port:       80,
				PidFile:    filepath.Join(base, "caddy", "caddy.pid"),
				LogFile:    filepath.Join(base, "caddy", "caddy.log"),
				Args: []string{
					"run",
					"--config", filepath.Join(base, "caddy", "Caddyfile"),
					"--pidfile", filepath.Join(base, "caddy", "caddy.pid"),
				},
			},
			"postgresql": {
				Name:       "PostgreSQL",
				BinaryPath: filepath.Join(base, "postgresql", "bin", binExe("postgres")),
				ConfigPath: filepath.Join(base, "postgresql", "data"),
				Port:       5432,
				PidFile:    filepath.Join(base, "postgresql", "data", "postmaster.pid"),
				LogFile:    filepath.Join(base, "postgresql", "data", "postgresql.log"),
				Args:       []string{"-D", filepath.Join(base, "postgresql", "data")},
			},

			"mysql": {
				Name:       "MySQL",
				BinaryPath: filepath.Join(base, "mysql", "bin", binExe("mysqld")),
				ConfigPath: filepath.Join(base, "mysql", "my.cnf"),
				Port:       3306,
				PidFile:    filepath.Join(base, "mysql", "data", "mysql.pid"),
				LogFile:    filepath.Join(base, "mysql", "data", "mysql-error.log"),
				Args: []string{
					"--defaults-file=" + filepath.Join(base, "mysql", "my.cnf"),
					"--pid-file=" + filepath.Join(base, "mysql", "data", "mysql.pid"),
				},
			},

			"php": {
				Name:       "PHP-FPM",
				BinaryPath: filepath.Join(base, "php", binExe("php-fpm")),
				ConfigPath: filepath.Join(base, "php", "php-fpm.conf"),
				Port:       9000,
				PidFile:    filepath.Join(base, "php", "php-fpm.pid"),
				LogFile:    filepath.Join(base, "php", "php-fpm.log"),
				Args: []string{
					"--fpm-config", filepath.Join(base, "php", "php-fpm.conf"),
					"--pid", filepath.Join(base, "php", "php-fpm.pid"),
				},
			},
		},
	}
}

func binExe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".luna")
}

func ConfigPath() string {
	return filepath.Join(BaseDir(), "luna.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o644)
}

func BinExe(name string) string {
	return binExe(name)
}
