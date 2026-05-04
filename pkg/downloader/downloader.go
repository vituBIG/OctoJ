// Package downloader provides utilities for downloading files with progress reporting.
package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Options configures a download operation.
type Options struct {
	// URL is the source URL to download.
	URL string
	// DestPath is the destination file path.
	DestPath string
	// UserAgent is the HTTP User-Agent header value.
	UserAgent string
	// ShowProgress enables the progress bar.
	ShowProgress bool
}

// Download fetches a file from a URL to a local path.
// If the destination file already exists and has a non-zero size, the download is skipped.
func Download(ctx context.Context, opts Options) error {
	if opts.UserAgent == "" {
		opts.UserAgent = "octoj/0.1.0 (https://github.com/OctavoBit/octoj)"
	}

	// Skip if already downloaded
	if info, err := os.Stat(opts.DestPath); err == nil && info.Size() > 0 {
		log.Debug().Str("path", opts.DestPath).Msg("file already downloaded, skipping")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(opts.DestPath), 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", opts.UserAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, opts.URL)
	}

	tmpPath := opts.DestPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	var reader io.Reader = resp.Body

	if opts.ShowProgress {
		p := mpb.New(mpb.WithWidth(60))
		total := resp.ContentLength
		fileName := filepath.Base(opts.DestPath)

		bar := p.AddBar(total,
			mpb.PrependDecorators(
				decor.Name("Downloading "),
				decor.Name(fileName, decor.WCSyncSpace),
			),
			mpb.AppendDecorators(
				decor.OnComplete(decor.EwmaETA(decor.ET_STYLE_GO, 30), "done"),
				decor.Name(" ] "),
				decor.CountersKibiByte("% .2f / % .2f"),
			),
		)
		proxyReader := bar.ProxyReader(resp.Body)
		defer proxyReader.Close()
		reader = proxyReader

		_, err = io.Copy(out, reader)
		p.Wait()
	} else {
		_, err = io.Copy(out, reader)
	}

	out.Close()

	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download failed: %w", err)
	}

	if err := os.Rename(tmpPath, opts.DestPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	return nil
}

// FetchText performs an HTTP GET and returns the response body as a string.
func FetchText(ctx context.Context, url, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if userAgent == "" {
		userAgent = "octoj/0.1.0"
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
