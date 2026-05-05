```
   ___  ___ _        _
  / _ \/ __| |_ ___ | |
 | (_) \__ \  _/ _ \| |
  \___/|___/\__\___/|_|
```

# OctoJ — Java JDK Version Manager

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/vituBIG/OctoJ/releases)
[![Build](https://github.com/vituBIG/OctoJ/actions/workflows/build.yml/badge.svg)](https://github.com/vituBIG/OctoJ/actions/workflows/build.yml)

**OctoJ** is a fast, multi-platform Java JDK version manager inspired by `nvm`, `jabba`, and `sdkman`.
Install, switch, and manage multiple JDK versions across Temurin, Corretto, Zulu, and Liberica — all from a single CLI tool.

---

## Quick Install

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/vituBIG/OctoJ/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
& ([scriptblock]::Create((New-Object Net.WebClient).DownloadString('https://raw.githubusercontent.com/vituBIG/OctoJ/main/scripts/install.ps1')))
```

> No administrator privileges required on any platform!

---

## First-time setup

Run this once after installing OctoJ. It sets `OCTOJ_HOME`, `JAVA_HOME`, and
places OctoJ at the **front** of your user PATH so it wins over any other Java
installation:

```bash
octoj init --apply
```

> **Windows**: restart your terminal (or open a new one) after running this
> command for the PATH changes to take effect.

Verify everything is correct:

```bash
octoj doctor
```

If `doctor` reports that another Java is shadowing OctoJ (e.g. Eclipse Adoptium
or Oracle JDK installed system-wide), remove its entry from
**System Properties → Environment Variables → System variables → Path** and
restart your terminal.

---

## Daily workflow

```bash
# Search available versions (all providers)
octoj search 21

# Search a specific provider
octoj search temurin 21
octoj search corretto@17

# Install a JDK (Temurin is the default provider)
octoj install 21
octoj install temurin@21.0.3+9   # exact version

# Install from another provider
octoj install corretto@17
octoj install zulu@11
octoj install liberica@21

# Switch the active JDK
octoj use temurin@21
octoj use corretto@17

# Confirm what is active
octoj current
java --version

# List all installed JDKs
octoj installed

# Remove a JDK
octoj uninstall temurin@17.0.18+8

# Clean download cache
octoj cache clean

# Remove OctoJ entirely (all JDKs + config)
octoj purge
```

---

## Commands

| Command | Description |
|---------|-------------|
| `octoj init` | Preview environment setup changes |
| `octoj init --apply` | Apply environment setup (OCTOJ_HOME, JAVA_HOME, PATH) |
| `octoj search <version>` | Search available JDK releases |
| `octoj search <provider> <version>` | Search from a specific provider |
| `octoj install <version>` | Install Temurin JDK (default provider) |
| `octoj install <provider>@<version>` | Install from a specific provider |
| `octoj use <provider>@<version>` | Activate an installed JDK version |
| `octoj current` | Show the currently active JDK |
| `octoj installed` | List all installed JDK versions |
| `octoj uninstall <provider>@<version>` | Remove an installed JDK |
| `octoj env` | Show OctoJ environment variables |
| `octoj doctor` | Diagnose the OctoJ installation |
| `octoj cache clean` | Clean the download cache |
| `octoj cache list` | List cached downloads |
| `octoj self-update` | Update OctoJ to the latest version |
| `octoj purge` | Completely remove OctoJ and all installed JDKs |
| `octoj purge --keep-binary` | Remove data but keep the `octoj` binary |
| `octoj purge --force` | Skip confirmation prompt (dangerous!) |

### Global Flags

| Flag | Description |
|------|-------------|
| `--verbose` / `-v` | Enable verbose logging |
| `--log-level <level>` | Set log level (trace, debug, info, warn, error) |
| `--config <file>` | Use a custom config file |
| `--os <os>` | Override target OS (windows, linux, darwin) |
| `--arch <arch>` | Override target architecture (x64, arm64) |

---

## Supported Providers

| Provider | Short Name | API | Notes |
|----------|------------|-----|-------|
| [Eclipse Temurin](https://adoptium.net) | `temurin` | Adoptium API v3 | **Default provider** |
| [Amazon Corretto](https://aws.amazon.com/corretto/) | `corretto` | Direct download | Versions: 8, 11, 17, 21 |
| [Azul Zulu](https://www.azul.com/downloads/) | `zulu` | Azul Metadata API v1 | Community edition |
| [BellSoft Liberica](https://bell-sw.com/libericajdk/) | `liberica` | BellSoft API v1 | Full JDK |

### Provider Syntax

```bash
# These all install the same thing (Temurin is default):
octoj install 21
octoj install temurin@21

# Provider-specific:
octoj install corretto@17
octoj install zulu@11
octoj install liberica@21

# Search syntax:
octoj search temurin 21
octoj search corretto@17
```

---

## Platform Support

| Platform | Architecture | Supported |
|----------|-------------|-----------|
| Windows 10/11 | x64 | Yes |
| Linux | x64, arm64 | Yes |
| macOS | x64 (Intel), arm64 (Apple Silicon) | Yes |

### Platform Notes

- **Windows**: Uses directory junctions (no admin required). Modifies `HKCU\Environment`.
- **Linux**: Detects bash/zsh/fish and modifies the appropriate rc file.
- **macOS**: Detects bash/zsh/fish. Adds config to `~/.zshrc`, `~/.bash_profile`, or fish config.

---

## Directory Layout

```
~/.octoj/                    # OCTOJ_HOME
├── config.json              # OctoJ configuration
├── jdks/                    # Installed JDKs
│   ├── temurin/
│   │   ├── 21.0.3+9/        # Temurin JDK 21
│   │   └── 17.0.11+9/       # Temurin JDK 17
│   ├── corretto/
│   │   └── 17.latest/
│   └── zulu/
│       └── 11.0.23+9/
├── current/                 # Symlink/junction → active JDK (= JAVA_HOME)
├── downloads/               # Temporary download cache
├── cache/                   # Metadata cache
├── bin/                     # OctoJ binaries
└── logs/                    # Log files
```

---

## Architecture

```
┌─────────────────────────────────────────┐
│              OctoJ CLI                  │
│          (cobra commands)               │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────▼─────────────┐
    │        Core Domain        │
    │  ┌──────────────────────┐ │
    │  │  Provider Registry   │ │
    │  │  (temurin|corretto   │ │
    │  │   zulu|liberica)     │ │
    │  └──────────────────────┘ │
    │  ┌──────────────────────┐ │
    │  │     Installer        │ │
    │  │  download→verify→    │ │
    │  │  extract→activate    │ │
    │  └──────────────────────┘ │
    │  ┌──────────────────────┐ │
    │  │  Environment Manager │ │
    │  │  (Windows|Unix)      │ │
    │  └──────────────────────┘ │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │     Storage Layer         │
    │  ~/.octoj/ filesystem     │
    └───────────────────────────┘
```

---

## Roadmap

- [x] Core provider framework
- [x] Temurin (Adoptium) provider
- [x] Amazon Corretto provider
- [x] Azul Zulu provider
- [x] BellSoft Liberica provider
- [x] Windows environment setup (registry, no admin)
- [x] Linux/macOS shell configuration
- [x] Download with progress bar
- [x] SHA-256 checksum verification
- [x] Self-update mechanism
- [ ] `.octoj-version` file support (per-directory JDK version)
- [ ] GraalVM / OpenJ9 providers
- [ ] Shell completion (bash, zsh, fish, PowerShell)
- [ ] Offline mode
- [ ] JDK update notifications
- [ ] TUI (interactive selection)

---

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Clone and build
git clone https://github.com/vituBIG/OctoJ.git
cd octoj
go mod download
go build ./cmd/octoj

# Run tests
go test ./...

# Run the built binary
./octoj doctor
```

---

## License

[MIT License](LICENSE) — Copyright (c) 2026 vituBIG

---

Made with love by [OctavoBit](https://github.com/OctavoBit)