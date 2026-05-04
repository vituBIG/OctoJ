package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/spf13/cobra"
)

func newInstalledCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installed",
		Short: "List all installed JDK versions",
		Long:  `List all JDK versions currently installed and managed by OctoJ.`,
		Example: `  octoj installed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstalled()
		},
	}

	return cmd
}

func runInstalled() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	installed, err := store.ListInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed JDKs: %w", err)
	}

	if len(installed) == 0 {
		fmt.Println("No JDK versions installed.")
		fmt.Println("Run `octoj install <version>` to install a JDK.")
		return nil
	}

	// Determine which is currently active
	currentTarget := ""
	currentPath := store.CurrentPath()
	if info, err := os.Lstat(currentPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(currentPath); err == nil {
				currentTarget = target
			}
		} else {
			currentTarget = currentPath
		}
	}

	fmt.Printf("Installed JDK versions:\n\n")
	fmt.Printf("  %-8s %-15s %-20s\n", "ACTIVE", "PROVIDER", "VERSION")
	fmt.Printf("  %-8s %-15s %-20s\n", strings.Repeat("-", 6), strings.Repeat("-", 15), strings.Repeat("-", 20))

	for _, jdk := range installed {
		active := "  "
		jdkPath := store.JDKPath(jdk.Provider, jdk.FullVersion)
		if currentTarget != "" && (currentTarget == jdkPath || filepath.Clean(currentTarget) == filepath.Clean(jdkPath)) {
			active = "* "
		}
		fmt.Printf("  %-8s %-15s %-20s\n", active, jdk.Provider, jdk.FullVersion)
	}

	fmt.Println()
	fmt.Println("* = currently active")
	fmt.Println()
	fmt.Printf("Total: %d JDK(s) installed\n", len(installed))

	return nil
}
