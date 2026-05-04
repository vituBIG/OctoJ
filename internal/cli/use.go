package cli

import (
	"fmt"
	"strings"

	"github.com/OctavoBit/octoj/internal/installer"
	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [provider@]<version>",
		Short: "Activate an installed JDK version",
		Long: `Switch the active JDK version by updating the 'current' symlink/junction.

The specified version must already be installed. Use 'octoj installed' to see
installed versions, and 'octoj install' to install new ones.`,
		Example: `  octoj use temurin@21      # activate Temurin JDK 21
  octoj use corretto@17     # activate Corretto JDK 17
  octoj use 21              # activate JDK 21 (searches installed versions)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUse(args[0])
		},
	}

	return cmd
}

func runUse(arg string) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	providerName, version := parseProviderVersion(arg)

	// If only version was given (no provider), find the installed version
	if !strings.Contains(arg, "@") {
		installed, err := store.ListInstalled()
		if err != nil {
			return fmt.Errorf("failed to list installed versions: %w", err)
		}

		var matches []storage.InstalledJDK
		for _, jdk := range installed {
			if jdk.Version == version || strings.HasPrefix(jdk.FullVersion, version) {
				matches = append(matches, jdk)
			}
		}

		switch len(matches) {
		case 0:
			return fmt.Errorf("no installed JDK found for version %q — run `octoj install %s` first", version, version)
		case 1:
			providerName = matches[0].Provider
			version = matches[0].FullVersion
		default:
			fmt.Printf("Multiple installed versions match %q:\n\n", version)
			for i, m := range matches {
				fmt.Printf("  [%d] %s@%s\n", i+1, m.Provider, m.FullVersion)
			}
			fmt.Println("\nSpecify provider: e.g., `octoj use temurin@21`")
			return nil
		}
	}

	registry := providers.NewRegistry()
	if _, err := registry.Get(providerName); err != nil {
		return fmt.Errorf("unknown provider %q: %w", providerName, err)
	}

	release := &providers.JDKRelease{
		Provider:    providerName,
		Version:     version,
		FullVersion: version,
	}

	inst := installer.New(store)

	// Check it's actually installed
	jdkPath := store.JDKPath(providerName, version)
	if !store.DirExists(jdkPath) {
		// Try to find by major version
		installed, err := store.ListInstalled()
		if err != nil {
			return err
		}
		for _, jdk := range installed {
			if jdk.Provider == providerName && (jdk.Version == version || strings.HasPrefix(jdk.FullVersion, version)) {
				release.FullVersion = jdk.FullVersion
				break
			}
		}
		if !store.DirExists(store.JDKPath(providerName, release.FullVersion)) {
			return fmt.Errorf("%s@%s is not installed — run `octoj install %s@%s` first",
				providerName, version, providerName, version)
		}
	}

	log.Info().
		Str("provider", providerName).
		Str("version", release.FullVersion).
		Msg("activating JDK")

	if err := inst.Activate(release); err != nil {
		return fmt.Errorf("failed to activate: %w", err)
	}

	fmt.Printf("Now using %s@%s\n", providerName, release.FullVersion)
	fmt.Println("JAVA_HOME is pointing to the selected JDK.")

	return nil
}
