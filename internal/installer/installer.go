// Package installer handles JDK download, verification, extraction and activation.
package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/OctavoBit/octoj/internal/storage"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Installer manages JDK installation operations.
type Installer struct {
	store *storage.Storage
}

// New creates a new Installer backed by the given storage.
func New(store *storage.Storage) *Installer {
	return &Installer{store: store}
}

// Install downloads, verifies, extracts, and records a JDK release.
func (inst *Installer) Install(ctx context.Context, release *providers.JDKRelease) error {
	// Step 1: Download
	archivePath, err := inst.download(ctx, release)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		// Clean up the downloaded archive after extraction
		if err := os.Remove(archivePath); err != nil {
			log.Debug().Err(err).Str("path", archivePath).Msg("failed to remove archive after installation")
		}
	}()

	// Step 2: Verify checksum
	if release.Checksum != "" || release.ChecksumLink != "" {
		fmt.Print("Verifying checksum... ")
		expected := release.Checksum
		if release.ChecksumLink != "" {
			if fetched, err := fetchChecksumFromURL(ctx, release.ChecksumLink); err == nil {
				expected = fetched
			} else {
				log.Debug().Err(err).Msg("failed to fetch checksum_link, falling back to API checksum")
			}
		}
		if err := inst.verifyChecksum(archivePath, expected, release.ChecksumType); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("OK")
	}

	// Step 3: Extract
	destDir := inst.store.JDKPath(release.Provider, release.FullVersion)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create JDK directory: %w", err)
	}

	fmt.Print("Extracting... ")
	if err := inst.extract(archivePath, destDir, release); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("extraction failed: %w", err)
	}
	fmt.Println("done")

	// Step 4: Validate bin/java exists
	javaExe := filepath.Join(destDir, "bin", "java")
	if runtime.GOOS == "windows" {
		javaExe += ".exe"
	}
	if _, err := os.Stat(javaExe); err != nil {
		return fmt.Errorf("installation seems incomplete: java binary not found at %s", javaExe)
	}

	log.Info().
		Str("provider", release.Provider).
		Str("version", release.FullVersion).
		Str("path", destDir).
		Msg("JDK installed successfully")

	return nil
}

// download fetches the JDK archive with a progress bar.
func (inst *Installer) download(ctx context.Context, release *providers.JDKRelease) (string, error) {
	downloadsDir := inst.store.DownloadsDir()
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create downloads directory: %w", err)
	}

	fileName := release.FileName
	if fileName == "" {
		parts := strings.Split(release.URL, "/")
		fileName = parts[len(parts)-1]
	}

	destPath := filepath.Join(downloadsDir, fileName)

	// If already downloaded, skip
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		log.Debug().Str("path", destPath).Msg("using cached download")
		fmt.Printf("Using cached download: %s\n", fileName)
		return destPath, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, release.URL)
	}

	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Progress bar
	p := mpb.New(mpb.WithWidth(60))
	total := resp.ContentLength

	bar := p.AddBar(total,
		mpb.PrependDecorators(
			decor.Name("Downloading "),
			decor.Name(fileName),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.EwmaETA(decor.ET_STYLE_GO, 30), "done"),
			decor.Name(" "),
			decor.CountersKibiByte("% .2f / % .2f"),
		),
	)

	proxyReader := bar.ProxyReader(resp.Body)
	defer proxyReader.Close()

	_, err = io.Copy(out, proxyReader)
	p.Wait()
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download interrupted: %w", err)
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close download file: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to finalize download: %w", err)
	}

	return destPath, nil
}

// fetchChecksumFromURL downloads a .sha256 file and returns the hash.
// Handles both "<hash>" and "<hash>  <filename>" (sha256sum) formats.
func fetchChecksumFromURL(ctx context.Context, link string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum URL returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}
	return strings.ToLower(fields[0]), nil
}

// verifyChecksum checks the SHA-256 (or MD5) checksum of a file.
func (inst *Installer) verifyChecksum(filePath, expectedChecksum, checksumType string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedChecksum) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actual)
	}

	return nil
}

// extract unpacks a .tar.gz or .zip archive into destDir.
func (inst *Installer) extract(archivePath, destDir string, release *providers.JDKRelease) error {
	if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return extractTarGz(archivePath, destDir)
	}
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractTarGz extracts a .tar.gz archive, stripping the top-level directory.
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	// Detect top-level directory name in the archive
	topDir := ""

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Strip top-level directory
		cleanName := filepath.Clean(header.Name)
		parts := strings.SplitN(filepath.ToSlash(cleanName), "/", 2)
		if topDir == "" {
			topDir = parts[0]
		}

		var relPath string
		if len(parts) > 1 {
			relPath = parts[1]
		} else {
			// This is the top-level directory itself
			continue
		}

		if relPath == "" {
			continue
		}

		targetPath := filepath.Join(destDir, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)|0o111); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			out.Close()

		case tar.TypeSymlink:
			os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				log.Debug().Err(err).Str("link", targetPath).Msg("failed to create symlink")
			}
		}
	}

	return nil
}

// extractZip extracts a .zip archive, stripping the top-level directory.
func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Detect top-level directory
	topDir := ""
	for _, f := range r.File {
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if topDir == "" && f.FileInfo().IsDir() {
			topDir = parts[0]
			break
		}
	}

	for _, f := range r.File {
		cleanName := filepath.ToSlash(f.Name)
		// Strip top-level directory
		if topDir != "" && strings.HasPrefix(cleanName, topDir+"/") {
			cleanName = cleanName[len(topDir)+1:]
		}

		if cleanName == "" {
			continue
		}

		targetPath := filepath.Join(destDir, filepath.FromSlash(cleanName))

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, f.Mode()|0o111); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// Activate creates or updates the 'current' symlink/junction to point to the given release.
func (inst *Installer) Activate(release *providers.JDKRelease) error {
	jdkPath := inst.store.JDKPath(release.Provider, release.FullVersion)
	currentPath := inst.store.CurrentPath()

	// Verify the JDK is actually installed there
	if _, err := os.Stat(jdkPath); err != nil {
		return fmt.Errorf("JDK not found at %s: %w", jdkPath, err)
	}

	// Remove existing current symlink/junction
	if _, err := os.Lstat(currentPath); err == nil {
		if err := removeJunction(currentPath); err != nil {
			return fmt.Errorf("failed to remove existing current link: %w", err)
		}
	}

	// Create new symlink/junction
	if err := createJunction(jdkPath, currentPath); err != nil {
		return fmt.Errorf("failed to create current link: %w", err)
	}

	// Create .cmd shims in the OctoJ bin dir so %OCTOJ_HOME%\bin in PATH
	// always resolves to the active JDK, regardless of other Java installations.
	if err := EnsureShims(inst.store.BinDir()); err != nil {
		log.Warn().Err(err).Msg("failed to create Java shims")
	}

	log.Debug().
		Str("provider", release.Provider).
		Str("version", release.FullVersion).
		Str("current", currentPath).
		Msg("activated JDK")

	inst.warnIfJavaShadowed()

	return nil
}

// warnIfJavaShadowed prints a warning when another Java installation in PATH would shadow OctoJ.
func (inst *Installer) warnIfJavaShadowed() {
	javaExe, err := exec.LookPath("java")
	if err != nil {
		return
	}
	octojBinDir := filepath.ToSlash(strings.ToLower(inst.store.BinDir()))
	currentBinDir := filepath.ToSlash(strings.ToLower(filepath.Join(inst.store.CurrentPath(), "bin")))
	javaLower := filepath.ToSlash(strings.ToLower(javaExe))

	if !strings.HasPrefix(javaLower, octojBinDir) && !strings.HasPrefix(javaLower, currentBinDir) {
		fmt.Printf("\nWARNING: 'java' in PATH resolves to %s\n", javaExe)
		fmt.Println("         This installation shadows OctoJ. To fix it:")
		fmt.Println("         1. Open System Properties → Environment Variables → System Variables → Path")
		fmt.Println("         2. Remove or disable the entry for the competing Java installation")
		fmt.Println("         3. Restart your terminal")
	}
}
