package cli

import (
	"fmt"
	"os"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Show OctoJ environment variables",
		Long:  `Display the environment variables configured and managed by OctoJ.`,
		Example: `  octoj env`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnv()
		},
	}

	return cmd
}

func runEnv() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	octojHome := store.Home()
	currentPath := store.CurrentPath()

	fmt.Println("OctoJ Environment Variables:")
	fmt.Println()

	// Show what OctoJ manages
	fmt.Printf("  OCTOJ_HOME  = %s\n", octojHome)
	fmt.Printf("  JAVA_HOME   = %s\n", currentPath)
	fmt.Printf("  PATH entry  = %s/bin\n", octojHome)
	fmt.Printf("  PATH entry  = %s/bin\n", currentPath)
	fmt.Println()

	// Show actual current environment
	fmt.Println("Current environment:")
	fmt.Println()

	envVars := []string{"OCTOJ_HOME", "JAVA_HOME", "PATH"}
	for _, v := range envVars {
		val := os.Getenv(v)
		if val == "" {
			val = "(not set)"
		}
		if v == "PATH" {
			// Truncate PATH for readability
			if len(val) > 80 {
				val = val[:80] + "..."
			}
		}
		fmt.Printf("  %-12s = %s\n", v, val)
	}

	fmt.Println()

	// Diagnose mismatches
	envOctojHome := os.Getenv("OCTOJ_HOME")
	if envOctojHome == "" {
		fmt.Println("  ! OCTOJ_HOME is not set. Run `octoj init --apply` to configure.")
	} else if envOctojHome != octojHome {
		fmt.Printf("  ! OCTOJ_HOME mismatch: env=%s, expected=%s\n", envOctojHome, octojHome)
	}

	envJavaHome := os.Getenv("JAVA_HOME")
	if envJavaHome == "" {
		fmt.Println("  ! JAVA_HOME is not set. Run `octoj init --apply` to configure.")
	}

	return nil
}
