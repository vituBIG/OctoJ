package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/spf13/cobra"
)

type checkResult struct {
	name    string
	ok      bool
	message string
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose OctoJ installation",
		Long:  `Run a series of checks to diagnose the OctoJ installation and environment.`,
		Example: `  octoj doctor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}

	return cmd
}

func runDoctor() error {
	fmt.Println("OctoJ Doctor — Running diagnostics...")
	fmt.Println()

	var checks []checkResult

	// Check 1: Platform detection
	det, err := platform.Detect()
	if err != nil {
		checks = append(checks, checkResult{"Platform detection", false, err.Error()})
	} else {
		checks = append(checks, checkResult{
			"Platform detection",
			true,
			fmt.Sprintf("OS=%s, Arch=%s, Go=%s", det.OS, det.Arch, runtime.Version()),
		})
	}

	// Check 2: Storage directories
	store, err := storage.New()
	if err != nil {
		checks = append(checks, checkResult{"Storage initialization", false, err.Error()})
	} else {
		if err := store.EnsureDirs(); err != nil {
			checks = append(checks, checkResult{"Storage directories", false, err.Error()})
		} else {
			checks = append(checks, checkResult{"Storage directories", true, store.Home()})
		}
	}

	// Check 3: OCTOJ_HOME environment variable
	octojHome := os.Getenv("OCTOJ_HOME")
	if octojHome == "" {
		checks = append(checks, checkResult{
			"OCTOJ_HOME env var",
			false,
			"not set — run `octoj init --apply`",
		})
	} else {
		checks = append(checks, checkResult{"OCTOJ_HOME env var", true, octojHome})
	}

	// Check 4: JAVA_HOME environment variable
	javaHome := os.Getenv("JAVA_HOME")
	if javaHome == "" {
		checks = append(checks, checkResult{
			"JAVA_HOME env var",
			false,
			"not set — run `octoj init --apply`",
		})
	} else {
		if _, err := os.Stat(javaHome); err != nil {
			checks = append(checks, checkResult{
				"JAVA_HOME env var",
				false,
				fmt.Sprintf("set to %s but directory does not exist", javaHome),
			})
		} else {
			checks = append(checks, checkResult{"JAVA_HOME env var", true, javaHome})
		}
	}

	// Check 5: java binary in PATH
	javaPath, err := exec.LookPath("java")
	if err != nil {
		checks = append(checks, checkResult{
			"java in PATH",
			false,
			"java not found in PATH",
		})
	} else {
		checks = append(checks, checkResult{"java in PATH", true, javaPath})
	}

	// Check 6: current symlink/junction
	if store != nil {
		currentPath := store.CurrentPath()
		if info, err := os.Lstat(currentPath); err != nil {
			checks = append(checks, checkResult{
				"current JDK symlink",
				false,
				"no active JDK — run `octoj use <version>`",
			})
		} else {
			var target string
			if info.Mode()&os.ModeSymlink != 0 {
				target, _ = os.Readlink(currentPath)
			} else {
				target = currentPath
			}
			// Verify target exists
			if _, err := os.Stat(filepath.Join(target, "bin", "java")); err != nil {
				checks = append(checks, checkResult{
					"current JDK symlink",
					false,
					fmt.Sprintf("points to %s but java binary not found", target),
				})
			} else {
				checks = append(checks, checkResult{"current JDK symlink", true, target})
			}
		}
	}

	// Check 7: installed JDKs count
	if store != nil {
		installed, err := store.ListInstalled()
		if err != nil {
			checks = append(checks, checkResult{"installed JDKs", false, err.Error()})
		} else {
			checks = append(checks, checkResult{
				"installed JDKs",
				len(installed) > 0,
				fmt.Sprintf("%d JDK(s) installed", len(installed)),
			})
		}
	}

	// Print results
	passed := 0
	failed := 0
	for _, c := range checks {
		icon := "✓"
		if !c.ok {
			icon = "✗"
			failed++
		} else {
			passed++
		}
		fmt.Printf("  [%s] %-25s %s\n", icon, c.name, c.message)
	}

	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		fmt.Println()
		fmt.Println("Run `octoj init --apply` to fix environment issues.")
		return fmt.Errorf("%d diagnostic check(s) failed", failed)
	}

	fmt.Println()
	fmt.Println("All checks passed! OctoJ is correctly configured.")

	return nil
}
