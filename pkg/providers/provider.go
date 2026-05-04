// Package providers defines the Provider interface and JDKRelease types for OctoJ.
package providers

import "context"

// JDKRelease represents a single JDK release available for download.
type JDKRelease struct {
	Provider     string
	Version      string // Major version, e.g., "21"
	FullVersion  string // Full version string, e.g., "21.0.3+9"
	OS           string // "windows", "linux", "darwin"
	Arch         string // "x64", "arm64"
	URL          string // Download URL
	Checksum     string // Expected checksum hex string
	ChecksumType string // "sha256", "md5", etc.
	FileName     string // Original archive filename
	Size         int64  // File size in bytes
}

// Provider is the interface that all JDK providers must implement.
type Provider interface {
	// Name returns the provider's canonical name (e.g., "temurin", "corretto").
	Name() string

	// Search returns available JDK releases matching the given version, OS and arch.
	// The version may be a major version ("21") or partial version string.
	Search(ctx context.Context, version string, os string, arch string) ([]JDKRelease, error)

	// GetRelease returns the best matching release for the given version, OS and arch.
	// Returns an error if no matching release is found.
	GetRelease(ctx context.Context, version string, os string, arch string) (*JDKRelease, error)
}
