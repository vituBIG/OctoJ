//go:build !windows

package installer

import "os"

// createJunction creates a symbolic link on Unix systems.
func createJunction(target, link string) error {
	return os.Symlink(target, link)
}

// removeJunction removes a symlink or directory on Unix systems.
func removeJunction(path string) error {
	return os.Remove(path)
}
