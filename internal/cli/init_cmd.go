package cli

import (
	"fmt"

	"github.com/OctavoBit/octoj/internal/env"
	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var apply bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize OctoJ — configure OCTOJ_HOME, JAVA_HOME, and PATH",
		Long: `Initialize OctoJ by setting up the required environment variables.

Without --apply, shows the changes that would be made.
With --apply, applies the changes to your shell configuration.

On Windows, modifies HKCU\Environment (no admin required).
On Linux/macOS, modifies shell rc files (e.g., ~/.bashrc, ~/.zshrc).`,
		Example: `  octoj init           # preview environment setup
  octoj init --apply   # apply environment setup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(apply)
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "apply the environment changes (modify shell/registry)")

	return cmd
}

func runInit(apply bool) error {
	// Ensure OctoJ storage directories exist
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := store.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create OctoJ directories: %w", err)
	}

	log.Info().Str("home", store.Home()).Msg("OctoJ home directory")

	manager, err := env.NewManager(store.Home())
	if err != nil {
		return fmt.Errorf("failed to create environment manager: %w", err)
	}

	changes, err := manager.Plan()
	if err != nil {
		return fmt.Errorf("failed to plan environment changes: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("OctoJ environment is already configured. No changes needed.")
		return nil
	}

	fmt.Println("The following environment changes will be made:")
	fmt.Println()
	for _, change := range changes {
		fmt.Printf("  %s\n", change)
	}
	fmt.Println()

	if !apply {
		fmt.Println("Run `octoj init --apply` to apply these changes.")
		return nil
	}

	if err := manager.Apply(); err != nil {
		return fmt.Errorf("failed to apply environment changes: %w", err)
	}

	fmt.Println("Environment configured successfully!")
	fmt.Println()
	manager.PrintRestartInstructions()

	return nil
}
