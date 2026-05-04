// Package env manages OctoJ environment variable configuration across platforms.
package env

import (
	"fmt"
	"runtime"
)

// Manager handles reading and applying environment variable changes for OctoJ.
type Manager interface {
	// Plan returns a list of human-readable descriptions of changes to be made.
	Plan() ([]string, error)
	// Apply applies the environment variable changes.
	Apply() error
	// PrintRestartInstructions prints platform-specific shell restart instructions.
	PrintRestartInstructions()
	// PlanRemoval returns a description of what Remove() will undo.
	PlanRemoval() ([]string, error)
	// Remove undoes all environment changes made by Apply().
	Remove() error
}

// NewManager creates the appropriate Manager for the current OS.
func NewManager(octojHome string) (Manager, error) {
	switch runtime.GOOS {
	case "windows":
		return newWindowsManager(octojHome), nil
	case "linux", "darwin":
		return newUnixManager(octojHome), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
