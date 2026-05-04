//go:build !windows

package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	blockStart = "# >>> octoj init >>>"
	blockEnd   = "# <<< octoj init <<<"
)

type unixManager struct {
	octojHome string
}

// NewManager returns the Unix (Linux/macOS) environment manager.
func NewManager(octojHome string) (Manager, error) {
	return &unixManager{octojHome: octojHome}, nil
}

func newUnixManager(octojHome string) Manager {
	return &unixManager{octojHome: octojHome}
}

func (m *unixManager) Plan() ([]string, error) {
	shell, rcFile, err := m.detectShell()
	if err != nil {
		return nil, err
	}

	var changes []string

	if m.hasBlock(rcFile) {
		changes = append(changes, fmt.Sprintf("OctoJ block already present in %s — no changes needed", rcFile))
		return changes, nil
	}

	changes = append(changes, fmt.Sprintf("Add OctoJ initialization block to %s (shell: %s)", rcFile, shell))
	changes = append(changes, fmt.Sprintf("  export OCTOJ_HOME=%q", m.octojHome))
	changes = append(changes, "  export JAVA_HOME=\"$OCTOJ_HOME/current\"")
	changes = append(changes, "  export PATH=\"$OCTOJ_HOME/bin:$JAVA_HOME/bin:$PATH\"")

	return changes, nil
}

func (m *unixManager) Apply() error {
	shell, rcFile, err := m.detectShell()
	if err != nil {
		return err
	}

	if m.hasBlock(rcFile) {
		fmt.Printf("OctoJ is already configured in %s\n", rcFile)
		return nil
	}

	var block string
	if shell == "fish" {
		block = m.fishBlock()
	} else {
		block = m.posixBlock()
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcFile, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s\n", block); err != nil {
		return fmt.Errorf("failed to write to %s: %w", rcFile, err)
	}

	fmt.Printf("Added OctoJ configuration to %s\n", rcFile)
	return nil
}

func (m *unixManager) PrintRestartInstructions() {
	shell, rcFile, _ := m.detectShell()

	fmt.Printf("Shell configuration updated in %s\n\n", rcFile)
	fmt.Println("To apply the changes in the current session, run:")
	fmt.Println()

	switch shell {
	case "fish":
		fmt.Printf("  source %s\n", rcFile)
	default:
		fmt.Printf("  source %s\n", rcFile)
	}

	fmt.Println()
	fmt.Println("Or open a new terminal window.")
}

// detectShell returns the current shell name, rc file path, and any error.
func (m *unixManager) detectShell() (shellName, rcFile string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	// Check SHELL environment variable
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// Try to detect from /proc on Linux
		if runtime.GOOS == "linux" {
			out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", os.Getppid()), "-o", "comm=").Output()
			if err == nil {
				shellPath = strings.TrimSpace(string(out))
			}
		}
	}

	shellName = filepath.Base(shellPath)

	switch shellName {
	case "zsh":
		rcFile = filepath.Join(home, ".zshrc")
	case "fish":
		rcFile = filepath.Join(home, ".config", "fish", "config.fish")
		if err := os.MkdirAll(filepath.Dir(rcFile), 0o755); err != nil {
			return "", "", fmt.Errorf("failed to create fish config directory: %w", err)
		}
	case "bash", "":
		// Default to bash
		shellName = "bash"
		if runtime.GOOS == "darwin" {
			rcFile = filepath.Join(home, ".bash_profile")
		} else {
			rcFile = filepath.Join(home, ".bashrc")
		}
	default:
		// Unknown shell, default to .profile
		shellName = shellName
		rcFile = filepath.Join(home, ".profile")
	}

	return shellName, rcFile, nil
}

// hasBlock checks if the OctoJ initialization block already exists in the file.
func (m *unixManager) hasBlock(rcFile string) bool {
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), blockStart)
}

// posixBlock returns the POSIX shell (bash/zsh) initialization block.
func (m *unixManager) posixBlock() string {
	return fmt.Sprintf(`%s
export OCTOJ_HOME="%s"
export JAVA_HOME="$OCTOJ_HOME/current"
export PATH="$OCTOJ_HOME/bin:$JAVA_HOME/bin:$PATH"
%s`, blockStart, m.octojHome, blockEnd)
}

// fishBlock returns the Fish shell initialization block.
func (m *unixManager) fishBlock() string {
	return fmt.Sprintf(`%s
set -gx OCTOJ_HOME "%s"
set -gx JAVA_HOME "$OCTOJ_HOME/current"
fish_add_path "$OCTOJ_HOME/bin"
fish_add_path "$JAVA_HOME/bin"
%s`, blockStart, m.octojHome, blockEnd)
}

// PlanRemoval describes what Remove() will do to the shell rc file.
func (m *unixManager) PlanRemoval() ([]string, error) {
	_, rcFile, err := m.detectShell()
	if err != nil {
		return nil, err
	}

	if !m.hasBlock(rcFile) {
		return []string{fmt.Sprintf("No OctoJ block found in %s — nothing to remove", rcFile)}, nil
	}

	return []string{fmt.Sprintf("Remove OctoJ initialization block from %s", rcFile)}, nil
}

// Remove strips the OctoJ initialization block from the shell rc file.
func (m *unixManager) Remove() error {
	_, rcFile, err := m.detectShell()
	if err != nil {
		return err
	}

	if !m.hasBlock(rcFile) {
		return nil
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	cleaned := removeBlock(string(content))

	if err := os.WriteFile(rcFile, []byte(cleaned), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", rcFile, err)
	}

	fmt.Printf("Removed OctoJ block from %s\n", rcFile)
	return nil
}

// removeBlock strips the octoj init block (and surrounding blank lines) from text.
func removeBlock(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skip := false

	for _, line := range lines {
		if strings.TrimSpace(line) == blockStart {
			skip = true
			// Also drop a preceding blank line if present
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			continue
		}
		if strings.TrimSpace(line) == blockEnd {
			skip = false
			continue
		}
		if !skip {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}
