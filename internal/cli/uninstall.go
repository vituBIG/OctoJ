package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall [provider@]<version>",
		Short: "Uninstall a JDK version",
		Long:  `Remove an installed JDK version from OctoJ management.`,
		Example: `  octoj uninstall temurin@21      # uninstall Temurin JDK 21
  octoj uninstall corretto@17     # uninstall Corretto JDK 17
  octoj uninstall temurin@21 -f   # uninstall without confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}

func runUninstall(arg string, force bool) error {
	providerName, version := parseProviderVersion(arg)

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Find the actual installed version (handle major version matching)
	installed, err := store.ListInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed versions: %w", err)
	}

	fullVersion := version
	found := false
	for _, jdk := range installed {
		if jdk.Provider == providerName && (jdk.FullVersion == version || jdk.Version == version) {
			fullVersion = jdk.FullVersion
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("%s@%s is not installed", providerName, version)
	}

	jdkPath := store.JDKPath(providerName, fullVersion)

	if !force {
		fmt.Printf("This will remove: %s@%s\n", providerName, fullVersion)
		fmt.Printf("Path: %s\n\n", jdkPath)
		fmt.Print("Are you sure? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Println("Uninstall cancelled.")
			return nil
		}
	}

	log.Info().
		Str("provider", providerName).
		Str("version", fullVersion).
		Str("path", jdkPath).
		Msg("uninstalling JDK")

	// Check if this is the currently active version and warn
	currentPath := store.CurrentPath()
	if target, err := os.Readlink(currentPath); err == nil {
		if strings.HasPrefix(target, jdkPath) || target == jdkPath {
			fmt.Println("Warning: This is the currently active JDK.")
			fmt.Println("After uninstall, run `octoj use <version>` to activate another version.")
		}
	}

	if err := store.RemoveJDK(providerName, fullVersion); err != nil {
		return fmt.Errorf("failed to remove JDK: %w", err)
	}

	fmt.Printf("Successfully uninstalled %s@%s\n", providerName, fullVersion)

	return nil
}
