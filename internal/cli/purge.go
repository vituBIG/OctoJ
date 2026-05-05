package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

	// Warn if Java processes are running — they may hold JDK files open.
	if procs := runningJavaProcesses(); len(procs) > 0 {
		fmt.Println("Warning: the following Java processes are running and may hold JDK files open.")
		fmt.Println("Close them before purging, or the data directory will be renamed instead of deleted.")
		for _, p := range procs {
			fmt.Printf("  • %s\n", p)
		}
		fmt.Println()
	}

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
	// On Windows the running executable is locked by the OS. If the binary lives
	// inside octojHome, move it out first so RemoveAll can proceed.
	if runtime.GOOS == "windows" {
		binaryPath = relocateBinaryOnWindows(binaryPath, octojHome, keepBinary)
	}

	fmt.Printf("Deleting %s...\n", octojHome)
	if _, err := os.Stat(octojHome); os.IsNotExist(err) {
		fmt.Println("  Directory does not exist, skipping.")
	} else if err := os.RemoveAll(octojHome); err != nil {
		// On Windows a third-party process (IDE, language server) may be
		// holding JDK files open. Rename the directory so it is out of the
		// way and a fresh install is not blocked, then ask the user to delete
		// it manually once those processes are closed.
		if runtime.GOOS == "windows" && isFileLocked(err) {
			oldDir := octojHome + ".old"
			_ = os.RemoveAll(oldDir) // clear a previous leftover if any
			if renameErr := os.Rename(octojHome, oldDir); renameErr == nil {
				fmt.Printf("  Warning: some JDK files are locked by a running process.\n")
				fmt.Printf("  Directory renamed to: %s\n", oldDir)
				fmt.Printf("  Delete it manually after closing all Java processes (IDEs, terminals with Java, etc.).\n")
			} else {
				return fmt.Errorf("failed to delete %s (files are locked by another process — close all Java/IDE processes and retry): %w", octojHome, err)
			}
		} else {
			return fmt.Errorf("failed to delete %s: %w", octojHome, err)
		}
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

// runningJavaProcesses returns a short description of each running java/javaw
// process. Returns nil if none are found or the check fails.
func runningJavaProcesses() []string {
	if runtime.GOOS != "windows" {
		out, err := exec.Command("pgrep", "-a", "java").Output()
		if err != nil {
			return nil
		}
		var procs []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				procs = append(procs, line)
			}
		}
		return procs
	}

	// Windows: use WMIC to list java/javaw processes
	out, err := exec.Command("wmic", "process", "where",
		"name='java.exe' or name='javaw.exe'",
		"get", "ProcessId,CommandLine", "/format:list").Output()
	if err != nil {
		return nil
	}

	var procs []string
	var pid, cmdLine string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CommandLine=") {
			cmdLine = strings.TrimPrefix(line, "CommandLine=")
		} else if strings.HasPrefix(line, "ProcessId=") {
			pid = strings.TrimPrefix(line, "ProcessId=")
		}
		if pid != "" && cmdLine != "" {
			desc := fmt.Sprintf("PID %s: %s", pid, cmdLine)
			if len(desc) > 120 {
				desc = desc[:117] + "..."
			}
			procs = append(procs, desc)
			pid, cmdLine = "", ""
		}
	}
	return procs
}

// isFileLocked returns true when err indicates a file is locked by another
// process (Windows error 32: ERROR_SHARING_VIOLATION).
func isFileLocked(err error) bool {
	return err != nil && strings.Contains(err.Error(), "being used by another process")
}

// relocateBinaryOnWindows moves the running binary out of octojHome before
// RemoveAll is called. Windows locks running executables, so they cannot be
// deleted (or included in a directory deletion) while the process is alive.
// Returns the new path of the binary (may be unchanged if relocation was not
// needed or failed).
func relocateBinaryOnWindows(binaryPath, octojHome string, keepBinary bool) string {
	absExe, err := filepath.Abs(binaryPath)
	if err != nil {
		return binaryPath
	}
	absHome, err := filepath.Abs(octojHome)
	if err != nil {
		return binaryPath
	}

	// Only act when the binary is actually inside octojHome.
	if !strings.HasPrefix(strings.ToLower(absExe), strings.ToLower(absHome+string(os.PathSeparator))) {
		return binaryPath
	}

	var destDir string
	if keepBinary {
		// User wants to keep the binary; move it to their home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			return binaryPath
		}
		destDir = home
	} else {
		// We only need it out of the way temporarily; use the system temp dir.
		destDir = os.TempDir()
	}

	dest := filepath.Join(destDir, filepath.Base(absExe))
	if err := os.Rename(absExe, dest); err != nil {
		log.Warn().Err(err).Str("src", absExe).Str("dst", dest).Msg("could not relocate binary before purge")
		return binaryPath
	}

	if keepBinary {
		fmt.Printf("  Binary moved to %s (kept as requested).\n", dest)
	}
	return dest
}

// removeBinary deletes the running binary.
// On Windows the binary is locked while running, so we rename it to .old and
// inform the user.
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
