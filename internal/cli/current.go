package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newCurrentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show the currently active JDK version",
		Long:  `Display information about the currently active JDK version managed by OctoJ.`,
		Example: `  octoj current`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCurrent()
		},
	}

	return cmd
}

func runCurrent() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	currentPath := store.CurrentPath()

	// Check if current symlink/junction exists
	info, err := os.Lstat(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No JDK is currently active.")
			fmt.Println("Run `octoj install <version>` to install a JDK.")
			return nil
		}
		return fmt.Errorf("failed to check current path: %w", err)
	}

	var target string
	if info.Mode()&os.ModeSymlink != 0 {
		target, err = os.Readlink(currentPath)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}
	} else {
		// On Windows it might be a junction directory
		target = currentPath
	}

	log.Debug().Str("target", target).Msg("current JDK target")

	// Try to determine provider and version from path
	jdksDir := store.JDKsDir()
	rel, err := filepath.Rel(jdksDir, target)
	if err == nil && !strings.HasPrefix(rel, "..") {
		parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(parts) == 2 {
			provider := parts[0]
			version := parts[1]
			fmt.Printf("Active JDK: %s@%s\n", provider, version)
		}
	}

	// Try to run java -version
	javaExe := filepath.Join(currentPath, "bin", "java")
	if isWindows() {
		javaExe += ".exe"
	}

	if _, err := os.Stat(javaExe); err == nil {
		out, err := exec.Command(javaExe, "-version").CombinedOutput()
		if err == nil {
			fmt.Println()
			fmt.Println("Java version information:")
			fmt.Println(strings.TrimSpace(string(out)))
		}
	}

	fmt.Println()
	fmt.Printf("JAVA_HOME: %s\n", currentPath)

	return nil
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
