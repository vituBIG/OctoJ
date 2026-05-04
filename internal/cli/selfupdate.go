package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const currentVersion = "0.1.0"
const githubRepo = "vituBIG/OctoJ"

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
	Body    string        `json:"body"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func newSelfUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update OctoJ to the latest version",
		Long: `Check for a newer version of OctoJ on GitHub and update if available.

Downloads the binary for the current platform and replaces the running executable.`,
		Example: `  octoj self-update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpdate()
		},
	}

	return cmd
}

func runSelfUpdate() error {
	fmt.Printf("OctoJ current version: v%s\n", currentVersion)
	fmt.Println("Checking for updates...")

	ctx := context.Background()

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == currentVersion {
		fmt.Printf("\nOctoJ is already up to date (v%s).\n", currentVersion)
		return nil
	}

	fmt.Printf("\nNew version available: %s\n", release.TagName)
	if release.Body != "" {
		fmt.Printf("\nRelease notes:\n%s\n", release.Body)
	}

	assetName := assetNameForPlatform()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s (expected asset name: %q)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	fmt.Printf("\nDownloading %s...\n", assetName)
	if err := downloadAndReplace(ctx, downloadURL, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", release.TagName)
	if runtime.GOOS == "windows" {
		fmt.Println("Restart your terminal to use the new version.")
	}

	return nil
}

func fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "octoj/"+currentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases published yet at github.com/%s", githubRepo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &release, nil
}

// assetNameForPlatform returns the expected release asset name for the current OS/arch.
func assetNameForPlatform() string {
	arch := runtime.GOARCH // amd64, arm64
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("octoj-windows-%s.exe", arch)
	default:
		return fmt.Sprintf("octoj-%s-%s", runtime.GOOS, arch)
	}
}

// downloadAndReplace downloads a binary from url and atomically replaces exePath.
func downloadAndReplace(ctx context.Context, url, exePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "octoj/"+currentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download server returned HTTP %d", resp.StatusCode)
	}

	tmpPath := exePath + ".new"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, copyErr := io.Copy(out, resp.Body)
	out.Close()
	if copyErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download interrupted: %w", copyErr)
	}

	// Windows: running binary is locked, rename it away first
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(exePath, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("could not move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not replace binary: %w", err)
	}

	return nil
}
