package cli

import (
	"context"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const currentVersion = "0.1.0"

func newSelfUpdateCmd() *cobra.Command {
	var prerelease bool

	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update OctoJ to the latest version",
		Long: `Check for a newer version of OctoJ and update if available.

Downloads the latest release from GitHub (vituBIG/OctoJ) and replaces
the current binary.`,
		Example: `  octoj self-update
  octoj self-update --prerelease`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdate(prerelease)
		},
	}

	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "include pre-release versions")

	return cmd
}

func runSelfUpdate(prerelease bool) error {
	fmt.Printf("OctoJ current version: v%s\n", currentVersion)
	fmt.Printf("Checking for updates (GOOS=%s, GOARCH=%s)...\n\n", runtime.GOOS, runtime.GOARCH)

	ctx := context.Background()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug("vituBIG/OctoJ"))
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		fmt.Println("No release found. OctoJ may not have any published releases yet.")
		return nil
	}

	log.Debug().Str("latest", latest.Version()).Msg("latest release detected")

	if !latest.GreaterThan(currentVersion) {
		fmt.Printf("OctoJ is already up to date (v%s).\n", currentVersion)
		return nil
	}

	fmt.Printf("New version available: v%s\n", latest.Version())
	fmt.Printf("Release notes: %s\n\n", latest.ReleaseNotes)
	fmt.Print("Updating OctoJ... ")

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("could not locate current executable: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println("done!")
	fmt.Printf("\nSuccessfully updated to v%s\n", latest.Version())
	fmt.Println("Restart octoj to use the new version.")

	return nil
}
