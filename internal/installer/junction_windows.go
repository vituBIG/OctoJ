//go:build windows

package installer

import (
	"fmt"
	"os"
	"os/exec"
)

// createJunction creates a directory junction on Windows (does not require admin).
func createJunction(target, link string) error {
	// Use mklink /J to create a directory junction (no admin required)
	cmd := exec.Command("cmd", "/C", "mklink", "/J", link, target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// removeJunction removes a directory junction on Windows.
func removeJunction(path string) error {
	// Check if it's a junction or symlink
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}

	// For directory junctions, use rmdir (not RemoveAll which would delete contents)
	cmd := exec.Command("cmd", "/C", "rmdir", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to os.Remove
		if removeErr := os.Remove(path); removeErr != nil {
			return fmt.Errorf("rmdir failed: %w\nOutput: %s", err, string(output))
		}
	}
	return nil
}
