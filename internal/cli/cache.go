package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage OctoJ download cache",
		Long:  `Commands for managing the OctoJ local download cache.`,
	}

	cacheCmd.AddCommand(newCacheCleanCmd())
	cacheCmd.AddCommand(newCacheListCmd())

	return cacheCmd
}

func newCacheCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove all cached downloads",
		Long:  `Delete all files in the OctoJ download cache to free disk space.`,
		Example: `  octoj cache clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCacheClean()
		},
	}
	return cmd
}

func newCacheListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cached downloads",
		Long:  `Show all files currently in the OctoJ download cache.`,
		Example: `  octoj cache list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCacheList()
		},
	}
	return cmd
}

func runCacheClean() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	cacheDir := store.CacheDir()
	downloadsDir := store.DownloadsDir()

	totalSize, err := dirSize(cacheDir)
	if err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("could not calculate cache size")
	}

	dlSize, err := dirSize(downloadsDir)
	if err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("could not calculate downloads size")
	}

	total := totalSize + dlSize

	log.Debug().
		Str("cache_dir", cacheDir).
		Str("downloads_dir", downloadsDir).
		Int64("total_bytes", total).
		Msg("cleaning cache")

	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean cache: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("failed to recreate cache dir: %w", err)
	}

	if err := os.RemoveAll(downloadsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean downloads: %w", err)
	}
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return fmt.Errorf("failed to recreate downloads dir: %w", err)
	}

	fmt.Printf("Cache cleaned. Freed approximately %s.\n", humanSize(total))

	return nil
}

func runCacheList() error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	dirs := []string{store.CacheDir(), store.DownloadsDir()}
	anyFile := false

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", dir, err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			anyFile = true
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			fmt.Printf("  %-50s %s\n", filepath.Join(dir, e.Name()), humanSize(size))
		}
	}

	if !anyFile {
		fmt.Println("Cache is empty.")
	}

	return nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
