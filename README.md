# 🌙 Luna

> A modern, cross-platform alternative to XAMPP. Manage your local development stack from a single CLI — no GUI, no bloat, no Docker required.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey?style=flat)]()
[![License](https://img.shields.io/badge/License-MIT-purple?style=flat)]()
[![Version](https://img.shields.io/badge/Version-0.1.0-7c6aff?style=flat)]()

---

## 📋 Daftar Isi

- [Tentang Luna](#-tentang-luna)
- [Stack](#-stack)
- [Instalasi](#-instalasi)
- [Penggunaan](#-penggunaan)
- [LunaBase](#-lunabase)
- [Konfigurasi](#-konfigurasi)
- [Struktur Project](#-struktur-project)
- [Build dari Source](#-build-dari-source)
- [Catatan Platform](#-catatan-platform)
- [Kontribusi](#-kontribusi)

---

## 🌙 Tentang Luna

Luna adalah CLI tool untuk manajemen local development environment yang terinspirasi dari XAMPP, namun didesain lebih modern dan ringan. Luna membundle semua service yang dibutuhkan dalam satu binary tunggal — tanpa perlu menginstall dependensi tambahan secara manual.

**Kenapa Luna?**

| | XAMPP | Luna |
|---|---|---|
| Interface | GUI | CLI |
| Web Server | Apache | Caddy |
| Database | MySQL | PostgreSQL + MySQL |
| PHP | ✅ | ✅ (PHP-FPM) |
| Cross-platform | ✅ | ✅ |
| Bundled binary | ✅ | ✅ |
| Dark mode UI | ❌ | ✅ (LunaBase) |
| Auto-download | ❌ | ✅ |

---

## 📦 Stack

Luna menggunakan stack berikut:

| Service | Versi | Port | Keterangan |
|---|---|---|---|
| [Caddy](https://caddyserver.com) | Latest | `:80` | Web server modern, pengganti Apache |
| [PostgreSQL](https://postgresql.org) | Latest | `:5432` | Database relasional utama |
| [MySQL](https://mysql.com) | 8.4.x | `:3306` | Database alternatif |
| [PHP-FPM](https://php.net) | 8.5.x | `:9000` | FastCGI PHP processor |
| [LunaBase](#-lunabase) | 1.0.0 | `/lunabase` | Database manager berbasis web |

---

## 🚀 Instalasi

### Prasyarat

- Go 1.22 atau lebih baru
- macOS, Linux, atau Windows
- Untuk macOS: Homebrew (diperlukan untuk PostgreSQL di macOS 12 dan MySQL)

### Download Binary

```bash
# Clone repository
git clone https://github.com/lunara-sh/luna.git
cd luna

# Build
go build -o luna ./cmd/luna

# Pindahkan ke PATH
mkdir -p ~/.luna/bin
cp luna ~/.luna/bin/

# Tambahkan ke PATH (tambahkan ke ~/.zshrc atau ~/.bashrc)
echo 'export PATH="$HOME/.luna/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Install Services

```bash
# Install semua service sekaligus
luna install

# Atau install satu per satu
luna install caddy
luna install postgresql
luna install mysql
luna install php
luna install php-cli
luna install adminer
```

> ⚠️ **macOS 12 (Monterey):** PostgreSQL akan otomatis diinstall via Homebrew karena binary pre-built membutuhkan macOS 13+. MySQL juga menggunakan Homebrew di semua versi macOS.

---

## 📖 Penggunaan

### Command Dasar

```bash
# Tampilkan status semua service
luna
luna status

# Start semua service
luna start

# Stop semua service
luna stop

# Restart semua service
luna restart
```

### Manajemen Service

```bash
# Start service tertentu
luna start caddy
luna start postgresql
luna start mysql
luna start php

# Stop service tertentu
luna stop caddy
luna stop mysql

# Restart service tertentu
luna restart postgresql
```

### Install

```bash
# Install semua service
luna install

# Install service tertentu
luna install caddy        # Caddy web server
luna install postgresql   # PostgreSQL database
luna install mysql        # MySQL database
luna install php          # PHP-FPM
luna install php-cli      # PHP CLI (untuk php -v, php artisan, dll)
luna install adminer      # Adminer database UI
luna install lunabase     # LunaBase database UI

# Cek update Caddy
luna install check
```

### Konfigurasi

```bash
# Tampilkan konfigurasi saat ini
luna config --show

# Tulis konfigurasi default ke disk
luna config --init

# Tampilkan versi Luna
luna version
```

### Contoh Output `luna status`

```
  ██╗     ██╗   ██╗███╗   ██╗ █████╗
  ██║     ██║   ██║████╗  ██║██╔══██╗
  ██║     ██║   ██║██╔██╗ ██║███████║
  ██║     ██║   ██║██║╚██╗██║██╔══██║
  ███████╗╚██████╔╝██║ ╚████║██║  ██║
  ╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝
  Local development environment manager

Service Status
──────────────────────────────────────────────────
  ●  Caddy          running  pid 1234    :80
  ●  PHP-FPM        running  pid 1235    :9000
  ●  PostgreSQL     running  pid 1236    :5432
  ●  MySQL          running  pid 1237    :3306

  Config stored at: ~/.luna/luna.json
```

---

## 🌙 LunaBase

LunaBase adalah database manager berbasis web yang dibangun khusus untuk Luna. Didesain modern dengan dark/light mode dan mendukung PostgreSQL maupun MySQL.

### Akses

Setelah service berjalan, buka browser dan akses:

```
http://localhost/lunabase/
```

### Fitur

- **Login** — Support PostgreSQL dan MySQL/MariaDB
- **Auto-create database** — Ketik nama database baru saat login, otomatis dibuat
- **Switch database** — Pindah antar database tanpa logout via dropdown sidebar
- **Buat database** — Modal form untuk membuat database baru
- **Hapus database** — Hapus database dengan konfirmasi (database default dilindungi)
- **List tables** — Tampilkan semua table dengan jumlah rows
- **View data** — Tampilkan isi table dengan pagination 25 rows per halaman
- **Buat table** — GUI form untuk membuat table baru dengan pilihan tipe data
- **SQL Editor** — Jalankan query SQL langsung, `Ctrl+Enter` untuk execute
- **Dark/Light mode** — Toggle tema, tersimpan di localStorage
- **Search table** — Filter table di dashboard

### Login PostgreSQL

| Field | Value |
|---|---|
| Driver | PostgreSQL |
| Host | localhost |
| Port | 5432 |
| Username | `nama_user_kamu` |
| Password | password kamu |
| Database | `postgres` (atau nama database lain) |

### Login MySQL

| Field | Value |
|---|---|
| Driver | MySQL / MariaDB |
| Host | localhost |
| Port | 3306 |
| Username | root |
| Password | (kosong) |
| Database | (kosong atau nama database) |

### Integrasi dengan Laravel

LunaBase mendukung PostgreSQL yang bisa digunakan langsung dengan Laravel:

```env
# PostgreSQL
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=nama_database
DB_USERNAME=nama_user
DB_PASSWORD=password

# MySQL
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=nama_database
DB_USERNAME=root
DB_PASSWORD=
```

---

## ⚙️ Konfigurasi

Konfigurasi Luna tersimpan di `~/.luna/luna.json`. File ini otomatis dibuat saat pertama kali menjalankan Luna.

### Contoh `luna.json`

```json
{
  "base_dir": "/Users/username/.luna",
  "services": {
    "caddy": {
      "name": "Caddy",
      "binary_path": "/Users/username/.luna/caddy/caddy",
      "config_path": "/Users/username/.luna/caddy/Caddyfile",
      "port": 80,
      "args": ["run", "--config", "/Users/username/.luna/caddy/Caddyfile", "--pidfile", "/Users/username/.luna/caddy/caddy.pid"],
      "pid_file": "/Users/username/.luna/caddy/caddy.pid",
      "log_file": "/Users/username/.luna/caddy/caddy.log"
    },
    "postgresql": {
      "name": "PostgreSQL",
      "binary_path": "/Users/username/.luna/postgresql/bin/postgres",
      "config_path": "/Users/username/.luna/postgresql/data",
      "port": 5432,
      "args": ["-D", "/Users/username/.luna/postgresql/data"],
      "pid_file": "/Users/username/.luna/postgresql/data/postmaster.pid",
      "log_file": "/Users/username/.luna/postgresql/data/postgresql.log"
    },
    "mysql": {
      "name": "MySQL",
      "binary_path": "/Users/username/.luna/mysql/bin/mysqld",
      "config_path": "/Users/username/.luna/mysql/my.cnf",
      "port": 3306,
      "args": ["--defaults-file=/Users/username/.luna/mysql/my.cnf"],
      "pid_file": "/Users/username/.luna/mysql/data/mysql.pid",
      "log_file": "/Users/username/.luna/mysql/data/mysql-error.log"
    },
    "php": {
      "name": "PHP-FPM",
      "binary_path": "/Users/username/.luna/php/php-fpm",
      "config_path": "/Users/username/.luna/php/php-fpm.conf",
      "port": 9000,
      "args": ["--fpm-config", "/Users/username/.luna/php/php-fpm.conf", "--pid", "/Users/username/.luna/php/php-fpm.pid"],
      "pid_file": "/Users/username/.luna/php/php-fpm.pid",
      "log_file": "/Users/username/.luna/php/php-fpm.log"
    }
  }
}
```

### Struktur Direktori `~/.luna`

```
~/.luna/
├── luna.json          # Konfigurasi utama
├── bin/               # Binary CLI Luna + PHP CLI
│   ├── luna
│   └── php
├── caddy/             # Caddy web server
│   ├── caddy          # Binary
│   ├── Caddyfile      # Konfigurasi
│   ├── caddy.pid
│   └── caddy.log
├── postgresql/        # PostgreSQL
│   ├── bin/
│   │   ├── postgres
│   │   ├── initdb
│   │   └── ...
│   └── data/          # Data directory
├── mysql/             # MySQL
│   ├── bin/
│   │   └── mysqld
│   ├── my.cnf
│   └── data/
├── php/               # PHP-FPM
│   ├── php-fpm
│   ├── php-fpm.conf
│   └── php-fpm.pid
└── www/               # Web root
    ├── index.html     # Landing page Luna
    ├── lunabase/      # LunaBase database UI
    │   └── index.php
    └── adminer/       # Adminer (opsional)
        └── index.php
```

---

## 🏗️ Struktur Project

```
luna/
├── cmd/
│   └── runtime.go             # Entry point CLI
├── internal/
│   ├── config/
│   │   └── config.go          # Konfigurasi & default paths
│   ├── process/
│   │   ├── process.go         # Process management (PID, port check)
│   │   ├── process_unix.go    # Unix-specific (SIGTERM)
│   │   └── process_windows.go # Windows-specific (Kill)
│   ├── service/
│   │   └── manager.go         # Service start/stop/restart/status
│   ├── install/
│   │   ├── install.go         # Download & install services
│   │   ├── fetch.go           # Fetch latest versions dari API
│   │   └── exec.go            # OS exec helpers
│   └── ui/
│       └── ui.go              # Terminal UI (colors, spinner, banner)
├── web/
│   └── lunabase/
│       └── index.php          # LunaBase database UI
├── go.mod
└── README.md
```

---

## 🔨 Build dari Source

### Prasyarat

- Go 1.22+
- Git

### Clone & Build

```bash
git clone https://github.com/lunara-sh/luna.git
cd luna
go mod tidy
go build -o luna ./cmd/luna
```

### Cross-compile

```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o luna-darwin-amd64 ./cmd/luna

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o luna-darwin-arm64 ./cmd/luna

# Linux
GOOS=linux GOARCH=amd64 go build -o luna-linux-amd64 ./cmd/luna

# Windows
GOOS=windows GOARCH=amd64 go build -o luna-windows-amd64.exe ./cmd/luna
```

---

## 🖥️ Catatan Platform

### macOS

| Versi | PostgreSQL | MySQL | Caddy | PHP-FPM |
|---|---|---|---|---|
| macOS 13+ (Ventura) | Binary bundled | Homebrew | Binary bundled | Binary bundled |
| macOS 12 (Monterey) | Homebrew fallback | Homebrew | Binary bundled | Binary bundled |

**Setup PATH di macOS:**

```bash
echo 'export PATH="$HOME/.luna/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Linux

Semua service menggunakan pre-built binary. Pastikan `tar` dan `xz-utils` terinstall untuk ekstraksi MySQL:

```bash
sudo apt install xz-utils   # Debian/Ubuntu
sudo yum install xz         # CentOS/RHEL
```

### Windows

Semua service menggunakan pre-built binary (`.zip`). Tambahkan `%USERPROFILE%\.luna\bin` ke system PATH.

---

## 🤝 Kontribusi

Pull request dan issue sangat diterima!

1. Fork repository
2. Buat branch baru (`git checkout -b feat/fitur-baru`)
3. Commit perubahan (`git commit -m 'feat: tambah fitur baru'`)
4. Push ke branch (`git push origin feat/fitur-baru`)
5. Buat Pull Request

### Conventional Commits

Luna menggunakan format [Conventional Commits](https://www.conventionalcommits.org):

```
feat:     fitur baru
fix:      bug fix
chore:    maintenance (update deps, dll)
docs:     perubahan dokumentasi
refactor: refactor kode
style:    formatting
test:     penambahan test
```

---

## 📄 Lisensi

MIT License — bebas digunakan, dimodifikasi, dan didistribusikan.

---

<div align="center">
  <p>Dibuat dengan ❤️ oleh <a href="https://github.com/lunara-sh">lunara-sh</a></p>
  <p>🌙 <em>Luna — karena development harusnya sesimpel melihat bulan.</em></p>
</div>
