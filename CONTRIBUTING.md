# Contributing to OctoJ

Thank you for your interest in contributing to OctoJ! This document provides guidelines for contributing.

---

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). All contributors are expected to uphold it.

---

## How to Contribute

### Reporting Bugs

1. Search [existing issues](https://github.com/OctavoBit/octoj/issues) to avoid duplicates.
2. Use the [bug report template](.github/ISSUE_TEMPLATE.md).
3. Include `octoj doctor` output in your report.
4. Specify your OS, architecture, and OctoJ version.

### Requesting Features

1. Open an issue with the `feature request` label.
2. Describe the use case and expected behavior.
3. Be specific — what problem does this feature solve?

### Submitting Code

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/octoj.git
   cd octoj
   ```
3. **Create a branch** following the naming convention:
   ```bash
   git checkout -b feat/my-new-feature    # for new features
   git checkout -b fix/issue-123          # for bug fixes
   git checkout -b docs/improve-readme    # for documentation
   git checkout -b refactor/installer     # for refactoring
   ```
4. **Make your changes** — see the Development section below.
5. **Test your changes** thoroughly.
6. **Push** to your fork and open a **Pull Request**.

---

## Branch Naming Convention

| Type | Pattern | Example |
|------|---------|---------|
| Feature | `feat/<description>` | `feat/graalvm-provider` |
| Bug fix | `fix/<issue-or-description>` | `fix/windows-junction` |
| Documentation | `docs/<description>` | `docs/improve-windows-guide` |
| Refactoring | `refactor/<description>` | `refactor/installer` |
| Tests | `test/<description>` | `test/temurin-provider` |
| CI/CD | `ci/<description>` | `ci/add-arm64-matrix` |

---

## Development Setup

### Prerequisites

- Go 1.22 or higher
- Git
- A terminal (bash, zsh, PowerShell, etc.)

### Build

```bash
# Clone the repo
git clone https://github.com/OctavoBit/octoj.git
cd octoj

# Download dependencies
go mod download

# Build
go build ./cmd/octoj

# Build for a specific platform
GOOS=linux GOARCH=amd64 go build -o octoj-linux-amd64 ./cmd/octoj
GOOS=windows GOARCH=amd64 go build -o octoj-windows-amd64.exe ./cmd/octoj
GOOS=darwin GOARCH=arm64 go build -o octoj-darwin-arm64 ./cmd/octoj
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a specific test
go test -run TestTemurinSearch ./pkg/providers/temurin/...
```

### Lint

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run ./...
```

---

## Project Structure

```
octoj/
├── cmd/octoj/           # Main entry point
├── internal/            # Private application code
│   ├── cli/             # Cobra command implementations
│   ├── config/          # Configuration management
│   ├── env/             # Environment variable setup
│   ├── installer/       # JDK download, verify, extract
│   ├── platform/        # OS/arch detection
│   └── storage/         # File system layout
├── pkg/                 # Public packages
│   ├── downloader/      # HTTP download utilities
│   └── providers/       # JDK provider implementations
│       ├── corretto/
│       ├── liberica/
│       ├── temurin/
│       └── zulu/
├── scripts/             # Install scripts
├── docs/                # Documentation
└── .github/             # GitHub Actions, templates
```

---

## Adding a New Provider

1. Create a new package under `pkg/providers/<name>/`.
2. Implement the `providers.Provider` interface:
   ```go
   type Provider interface {
       Name() string
       Search(ctx context.Context, version, os, arch string) ([]JDKRelease, error)
       GetRelease(ctx context.Context, version, os, arch string) (*JDKRelease, error)
   }
   ```
3. Register the provider in `pkg/providers/registry.go`.
4. Add tests for `Search` and `GetRelease`.
5. Document the API in `docs/providers.md`.

---

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR.
- Include tests for new functionality.
- Update documentation if behavior changes.
- Ensure all CI checks pass.
- Write clear commit messages:
  ```
  feat: add GraalVM provider

  Implements the Provider interface for GraalVM CE via GitHub releases API.
  Supports Java 17, 21 on Linux/macOS/Windows.

  Closes #42
  ```
- Link related issues in the PR description.

---

## Commit Message Format

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `ci`, `chore`

Examples:
```
feat(providers): add GraalVM CE provider
fix(windows): handle spaces in USERPROFILE path
docs(readme): add fish shell configuration example
test(temurin): add search API integration test
ci: add arm64 build matrix
```

---

## License

By contributing to OctoJ, you agree that your contributions will be licensed under the [MIT License](LICENSE).
