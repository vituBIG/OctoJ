# Changelog

All notable changes to OctoJ will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned
- `.octoj-version` file support for per-directory JDK version pinning
- GraalVM provider
- Shell completion for bash, zsh, fish, and PowerShell
- Offline mode with cached manifests
- JDK update notifications

---

## [0.1.0] — 2026-05-04

### Added
- Initial release of OctoJ — multi-platform Java JDK version manager
- `octoj init` / `octoj init --apply` — configure OCTOJ_HOME, JAVA_HOME, and PATH
  - Windows: modifies `HKCU\Environment` (no administrator required)
  - Linux/macOS: adds initialization block to shell rc files (bash, zsh, fish)
- `octoj search [provider@]<version>` — search available JDK releases
- `octoj install [provider@]<version>` — download, verify, extract, and activate a JDK
  - SHA-256 checksum verification
  - Progress bar during download
  - Supports `.tar.gz` (Linux/macOS) and `.zip` (Windows) archives
- `octoj use [provider@]<version>` — switch the active JDK via symlink/junction
- `octoj current` — display the currently active JDK version
- `octoj installed` — list all installed JDK versions with active indicator
- `octoj uninstall [provider@]<version>` — remove an installed JDK
- `octoj env` — show OctoJ environment variables and detect misconfigurations
- `octoj doctor` — run diagnostics and report installation health
- `octoj cache clean` — clear the download cache
- `octoj cache list` — list cached download files
- `octoj self-update` — update the OctoJ binary to the latest GitHub release
- **Provider: Eclipse Temurin** (default) via Adoptium API v3
- **Provider: Amazon Corretto** via direct download URLs (versions 8, 11, 17, 21)
- **Provider: Azul Zulu** via Azul Metadata API v1
- **Provider: BellSoft Liberica** via BellSoft API v1
- Platform support: Windows (x64), Linux (x64, arm64), macOS (x64, arm64)
- Windows: directory junctions for `current/` (no admin required)
- Unix: symlinks for `current/`
- GitHub Actions: CI build matrix, release pipeline with checksums
- Install scripts: `scripts/install.sh` (Linux/macOS), `scripts/install.ps1` (Windows)

[Unreleased]: https://github.com/vituBIG/OctoJ/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/vituBIG/OctoJ/releases/tag/v0.1.0
