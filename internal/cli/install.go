package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/OctavoBit/octoj/internal/installer"
	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		osFlag   string
		archFlag string
		activate bool
	)

	cmd := &cobra.Command{
		Use:   "install [provider@]<version>",
		Short: "Install a JDK version",
		Long: `Install a JDK version from a provider.

If no provider is specified, uses Temurin (Eclipse Adoptium) by default.
The version can be a major version number (e.g., 21) or a specific release.`,
		Example: `  octoj install 21              # install latest Temurin JDK 21
  octoj install temurin@21      # install Temurin JDK 21 explicitly
  octoj install corretto@17     # install Amazon Corretto JDK 17
  octoj install zulu@11         # install Azul Zulu JDK 11
  octoj install liberica@21     # install BellSoft Liberica JDK 21`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			det, err := platform.Detect()
			if err != nil {
				return err
			}
			if osFlag != "" {
				det.OS = osFlag
			}
			if archFlag != "" {
				det.Arch = archFlag
			}
			return runInstall(args[0], det, activate)
		},
	}

	cmd.Flags().StringVar(&osFlag, "os", "", "target OS (windows, linux, darwin)")
	cmd.Flags().StringVar(&archFlag, "arch", "", "target architecture (x64, arm64)")
	cmd.Flags().BoolVar(&activate, "activate", true, "activate the installed version")

	return cmd
}

func runInstall(arg string, det *platform.Info, activate bool) error {
	providerName, version := parseProviderVersion(arg)

	log.Info().
		Str("provider", providerName).
		Str("version", version).
		Str("os", det.OS).
		Str("arch", det.Arch).
		Msg("installing JDK")

	registry := providers.NewRegistry()
	p, err := registry.Get(providerName)
	if err != nil {
		return fmt.Errorf("unknown provider %q: %w", providerName, err)
	}

	ctx := context.Background()
	release, err := p.GetRelease(ctx, version, det.OS, det.Arch)
	if err != nil {
		return fmt.Errorf("failed to find release: %w", err)
	}

	fmt.Printf("Installing %s JDK %s (%s/%s)...\n", release.Provider, release.FullVersion, det.OS, det.Arch)
	fmt.Printf("  Source: %s\n\n", release.URL)

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := store.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to ensure storage directories: %w", err)
	}

	inst := installer.New(store)
	if err := inst.Install(ctx, release); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("\nSuccessfully installed %s@%s\n", release.Provider, release.FullVersion)

	if activate {
		if err := inst.Activate(release); err != nil {
			log.Warn().Err(err).Msg("installed but could not activate — run `octoj use` to activate manually")
		} else {
			fmt.Printf("Activated %s@%s\n", release.Provider, release.FullVersion)
			fmt.Println("\nRestart your terminal or run `octoj env` to verify the setup.")
		}
	}

	return nil
}

// parseProviderVersion parses "provider@version" or just "version" (defaults to temurin).
func parseProviderVersion(arg string) (provider, version string) {
	if strings.Contains(arg, "@") {
		parts := strings.SplitN(arg, "@", 2)
		return strings.ToLower(parts[0]), parts[1]
	}
	return "temurin", arg
}
