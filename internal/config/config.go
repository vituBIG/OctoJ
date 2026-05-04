// Package config manages OctoJ configuration persistence.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config holds the OctoJ configuration.
type Config struct {
	mu sync.RWMutex

	path string
	data *configData
}

// configData is the serializable config structure.
type configData struct {
	Version         string              `json:"version"`
	DefaultProvider string              `json:"default_provider"`
	ActiveJDK       string              `json:"active_jdk,omitempty"`
	InstalledJDKs   []InstalledJDKEntry `json:"installed_jdks"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// InstalledJDKEntry represents a single installed JDK record.
type InstalledJDKEntry struct {
	Provider    string    `json:"provider"`
	Version     string    `json:"version"`
	FullVersion string    `json:"full_version"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installed_at"`
}

// New loads or creates a config at the given path.
func New(configPath string) (*Config, error) {
	c := &Config{
		path: configPath,
		data: &configData{
			Version:         "1",
			DefaultProvider: "temurin",
			InstalledJDKs:   []InstalledJDKEntry{},
			UpdatedAt:       time.Now(),
		},
	}

	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return c, nil
}

// load reads the config file from disk.
func (c *Config) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c.data)
}

// Save persists the config to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data.UpdatedAt = time.Now()

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(c.path, data, 0o644)
}

// DefaultProvider returns the configured default provider.
func (c *Config) DefaultProvider() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data.DefaultProvider == "" {
		return "temurin"
	}
	return c.data.DefaultProvider
}

// SetDefaultProvider sets the default provider.
func (c *Config) SetDefaultProvider(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.DefaultProvider = provider
}

// ActiveJDK returns the currently active JDK identifier.
func (c *Config) ActiveJDK() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data.ActiveJDK
}

// SetActiveJDK sets the active JDK identifier.
func (c *Config) SetActiveJDK(jdk string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.ActiveJDK = jdk
}

// AddInstalledJDK records a newly installed JDK.
func (c *Config) AddInstalledJDK(entry InstalledJDKEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove existing entry with same provider+fullVersion
	var filtered []InstalledJDKEntry
	for _, e := range c.data.InstalledJDKs {
		if !(e.Provider == entry.Provider && e.FullVersion == entry.FullVersion) {
			filtered = append(filtered, e)
		}
	}

	if entry.InstalledAt.IsZero() {
		entry.InstalledAt = time.Now()
	}

	c.data.InstalledJDKs = append(filtered, entry)
}

// RemoveInstalledJDK removes a JDK record from the config.
func (c *Config) RemoveInstalledJDK(provider, fullVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var filtered []InstalledJDKEntry
	for _, e := range c.data.InstalledJDKs {
		if !(e.Provider == provider && e.FullVersion == fullVersion) {
			filtered = append(filtered, e)
		}
	}
	c.data.InstalledJDKs = filtered
}

// InstalledJDKs returns all recorded installed JDKs.
func (c *Config) InstalledJDKs() []InstalledJDKEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]InstalledJDKEntry, len(c.data.InstalledJDKs))
	copy(result, c.data.InstalledJDKs)
	return result
}
