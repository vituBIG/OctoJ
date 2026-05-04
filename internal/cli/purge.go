package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/OctavoBit/octoj/internal/env"
	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newPurgeCmd() *cobra.Command {
	var (
		force      bool
		keepBinary bool
	)

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Completely remove OctoJ and all installed JDKs",
		Long: `Remove OctoJ from your system entirely.

This command will:
  1. Undo environment variable changes (OCTOJ_HOME, JAVA_HOME, PATH)
  2. Remove the OctoJ shell block from your rc file (Linux/macOS)
  3. Delete all installed JDKs and OctoJ data (~/.octoj/)
  4. Optionally delete the octoj binary itself

Nothing is deleted until you confirm.`,
		Example: `  octoj purge                  # interactive, asks for confirmation
  octoj purge --force           # skip confirmation (dangerous!)
  octoj purge --keep-binary     # remove data but keep the octoj binary`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPurge(force, keepBinary)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&keepBinary, "keep-binary", false, "delete data but keep the octoj binary")

	return cmd
}

func runPurge(force, keepBinary bool) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	octojHome := store.Home()

	envMgr, err := env.NewManager(octojHome)
	if err != nil {
		return fmt.Errorf("failed to initialize environment manager: %w", err)
	}

	binaryPath, _ := os.Executable()

	// --- Build the plan ---
	fmt.Println("OctoJ Purge — the following actions will be performed:")
	fmt.Println()

	envChanges, err := envMgr.PlanRemoval()
	if err != nil {
		log.Warn().Err(err).Msg("could not determine environment changes")
	} else {
		for _, c := range envChanges {
			fmt.Printf("  • %s\n", c)
		}
	}

	fmt.Printf("  • Delete OctoJ data directory: %s\n", octojHome)

	if !keepBinary {
		fmt.Printf("  • Delete OctoJ binary: %s\n", binaryPath)
	}

	fmt.Println()

	// --- Confirm ---
	if !force {
		fmt.Print("This action is IRREVERSIBLE. Type \"yes\" to continue: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "yes" {
			fmt.Println("Purge cancelled.")
			return nil
		}
		fmt.Println()
	}

	// --- Step 1: Remove environment variables ---
	fmt.Println("Removing environment configuration...")
	if err := envMgr.Remove(); err != nil {
		log.Warn().Err(err).Msg("failed to remove environment configuration (continuing)")
		fmt.Printf("  Warning: %v\n", err)
	} else {
		fmt.Println("  Done.")
	}

	// --- Step 2: Delete ~/.octoj/ ---
	fmt.Printf("Deleting %s...\n", octojHome)
	if _, err := os.Stat(octojHome); os.IsNotExist(err) {
		fmt.Println("  Directory does not exist, skipping.")
	} else if err := os.RemoveAll(octojHome); err != nil {
		return fmt.Errorf("failed to delete %s: %w", octojHome, err)
	} else {
		fmt.Println("  Done.")
	}

	// --- Step 3: Delete binary (optional) ---
	if !keepBinary {
		fmt.Printf("Deleting binary %s...\n", binaryPath)
		if err := removeBinary(binaryPath); err != nil {
			log.Warn().Err(err).Msg("failed to delete binary")
			fmt.Printf("  Warning: could not delete binary automatically.\n")
			fmt.Printf("  Remove it manually: %s\n", binaryPath)
		} else {
			fmt.Println("  Done.")
		}
	}

	// --- Done ---
	fmt.Println()
	fmt.Println("OctoJ has been removed from your system.")

	if runtime.GOOS != "windows" {
		fmt.Println("Open a new terminal (or run `source ~/.zshrc`) for the environment changes to take effect.")
	} else {
		fmt.Println("Open a new terminal for the environment changes to take effect.")
	}

	return nil
}

// removeBinary deletes the running binary.
// On Windows the binary is locked while running, so we schedule deletion via a
// helper trick: rename to a .old file and mark it for deletion on next boot,
// or just inform the user.
func removeBinary(binaryPath string) error {
	binaryPath = filepath.Clean(binaryPath)

	if runtime.GOOS == "windows" {
		// On Windows the running executable is locked; rename it so future
		// installs aren't blocked, then let the user know.
		oldPath := binaryPath + ".old"
		if err := os.Rename(binaryPath, oldPath); err != nil {
			return fmt.Errorf("could not rename binary (it may be locked): %w", err)
		}
		fmt.Printf("  Binary renamed to %s — delete it manually after closing this terminal.\n", oldPath)
		return nil
	}

	return os.Remove(binaryPath)
}
