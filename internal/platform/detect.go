// Package platform provides OS and architecture detection for OctoJ.
package platform

import (
	"fmt"
	"runtime"
)

// Info holds the detected or user-specified platform information.
type Info struct {
	OS   string // "windows", "linux", "darwin"
	Arch string // "x64", "arm64"
}

// Detect returns the current platform information based on runtime.GOOS and runtime.GOARCH.
func Detect() (*Info, error) {
	os, err := normalizeOS(runtime.GOOS)
	if err != nil {
		return nil, err
	}

	arch, err := normalizeArch(runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	return &Info{OS: os, Arch: arch}, nil
}

// normalizeOS converts runtime.GOOS values to OctoJ OS names.
func normalizeOS(goos string) (string, error) {
	switch goos {
	case "windows":
		return "windows", nil
	case "linux":
		return "linux", nil
	case "darwin":
		return "darwin", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}
}

// normalizeArch converts runtime.GOARCH values to OctoJ arch names.
func normalizeArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}
}

// AdoptiumOS converts OctoJ OS to Adoptium API OS string.
func (i *Info) AdoptiumOS() string {
	switch i.OS {
	case "darwin":
		return "mac"
	default:
		return i.OS
	}
}

// AdoptiumArch converts OctoJ arch to Adoptium API arch string.
func (i *Info) AdoptiumArch() string {
	switch i.Arch {
	case "arm64":
		return "aarch64"
	default:
		return "x64"
	}
}

// AzulOS converts OctoJ OS to Azul API OS string.
func (i *Info) AzulOS() string {
	switch i.OS {
	case "darwin":
		return "macos"
	default:
		return i.OS
	}
}

// AzulArch converts OctoJ arch to Azul API arch string.
func (i *Info) AzulArch() string {
	switch i.Arch {
	case "arm64":
		return "arm"
	default:
		return "x86"
	}
}

// BellSoftOS converts OctoJ OS to BellSoft API OS string.
func (i *Info) BellSoftOS() string {
	switch i.OS {
	case "darwin":
		return "macos"
	default:
		return i.OS
	}
}

// BellSoftArch converts OctoJ arch to BellSoft API arch string.
func (i *Info) BellSoftArch() string {
	switch i.Arch {
	case "arm64":
		return "aarch64"
	default:
		return "amd64"
	}
}

// CorrettoOS converts OctoJ OS to Corretto URL OS segment.
func (i *Info) CorrettoOS() string {
	switch i.OS {
	case "darwin":
		return "macosx"
	default:
		return i.OS
	}
}

// CorrettoArch converts OctoJ arch to Corretto URL arch segment.
func (i *Info) CorrettoArch() string {
	switch i.Arch {
	case "arm64":
		return "aarch64"
	default:
		return "x64"
	}
}

// ArchiveExt returns the expected archive extension for the current platform.
func (i *Info) ArchiveExt() string {
	if i.OS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// IsWindows returns true if the current platform is Windows.
func (i *Info) IsWindows() bool {
	return i.OS == "windows"
}
