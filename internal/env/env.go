// Package env manages OctoJ environment variable configuration across platforms.
package env

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

// NewManager is implemented per-platform in windows.go and unix.go.
