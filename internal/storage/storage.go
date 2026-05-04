// Package storage manages the OctoJ file system layout.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// InstalledJDK represents a discovered installed JDK.
type InstalledJDK struct {
	Provider    string
	Version     string
	FullVersion string
	Path        string
}

// Storage provides access to OctoJ's directory layout.
type Storage struct {
	home string
}

// New creates a new Storage rooted at the default OctoJ home directory.
// Windows: %USERPROFILE%\.octoj
// Linux/macOS: ~/.octoj
func New() (*Storage, error) {
	home, err := octojHome()
	if err != nil {
		return nil, fmt.Errorf("failed to determine OctoJ home: %w", err)
	}
	return &Storage{home: home}, nil
}

// NewWithHome creates a Storage rooted at a custom home directory.
func NewWithHome(home string) *Storage {
	return &Storage{home: home}
}

func octojHome() (string, error) {
	// Allow override via environment variable
	if h := os.Getenv("OCTOJ_HOME"); h != "" {
		return h, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userHome, ".octoj"), nil
}

// Home returns the root OctoJ home directory.
func (s *Storage) Home() string {
	return s.home
}

// ConfigPath returns the path to config.json.
func (s *Storage) ConfigPath() string {
	return filepath.Join(s.home, "config.json")
}

// CacheDir returns the path to the metadata cache directory.
func (s *Storage) CacheDir() string {
	return filepath.Join(s.home, "cache")
}

// DownloadsDir returns the path to the downloads directory.
func (s *Storage) DownloadsDir() string {
	return filepath.Join(s.home, "downloads")
}

// JDKsDir returns the path to the jdks directory.
func (s *Storage) JDKsDir() string {
	return filepath.Join(s.home, "jdks")
}

// CurrentPath returns the path to the 'current' symlink/junction.
func (s *Storage) CurrentPath() string {
	return filepath.Join(s.home, "current")
}

// BinDir returns the path to the OctoJ bin directory.
func (s *Storage) BinDir() string {
	return filepath.Join(s.home, "bin")
}

// LogsDir returns the path to the logs directory.
func (s *Storage) LogsDir() string {
	return filepath.Join(s.home, "logs")
}

// JDKPath returns the path where a specific JDK would be installed.
func (s *Storage) JDKPath(provider, version string) string {
	return filepath.Join(s.home, "jdks", provider, version)
}

// EnsureDirs creates all required OctoJ directories if they don't exist.
func (s *Storage) EnsureDirs() error {
	dirs := []string{
		s.home,
		s.CacheDir(),
		s.DownloadsDir(),
		s.JDKsDir(),
		s.BinDir(),
		s.LogsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// DirExists returns true if the given path exists and is a directory.
func (s *Storage) DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ListInstalled scans the jdks directory and returns all installed JDKs.
func (s *Storage) ListInstalled() ([]InstalledJDK, error) {
	jdksDir := s.JDKsDir()

	providers, err := os.ReadDir(jdksDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read JDKs directory: %w", err)
	}

	var installed []InstalledJDK

	for _, providerEntry := range providers {
		if !providerEntry.IsDir() {
			continue
		}
		providerName := providerEntry.Name()
		providerDir := filepath.Join(jdksDir, providerName)

		versions, err := os.ReadDir(providerDir)
		if err != nil {
			continue
		}

		for _, versionEntry := range versions {
			if !versionEntry.IsDir() {
				continue
			}
			fullVersion := versionEntry.Name()
			jdkPath := filepath.Join(providerDir, fullVersion)

			// Verify it looks like a JDK (has bin/java)
			javaExe := filepath.Join(jdkPath, "bin", "java")
			if runtime.GOOS == "windows" {
				javaExe += ".exe"
			}
			if _, err := os.Stat(javaExe); err != nil {
				continue
			}

			// Parse major version from full version
			majorVersion := parseMajorVersion(fullVersion)

			installed = append(installed, InstalledJDK{
				Provider:    providerName,
				Version:     majorVersion,
				FullVersion: fullVersion,
				Path:        jdkPath,
			})
		}
	}

	return installed, nil
}

// RemoveJDK removes a JDK directory and updates config.
func (s *Storage) RemoveJDK(provider, fullVersion string) error {
	jdkPath := s.JDKPath(provider, fullVersion)

	if _, err := os.Stat(jdkPath); os.IsNotExist(err) {
		return fmt.Errorf("JDK directory not found: %s", jdkPath)
	}

	return os.RemoveAll(jdkPath)
}

// parseMajorVersion extracts the major version number from a full version string.
// e.g., "21.0.3+9" → "21", "17.0.11+9" → "17"
func parseMajorVersion(fullVersion string) string {
	for i, c := range fullVersion {
		if c == '.' || c == '+' || c == '-' || c == '_' {
			if i > 0 {
				return fullVersion[:i]
			}
			break
		}
	}
	return fullVersion
}
