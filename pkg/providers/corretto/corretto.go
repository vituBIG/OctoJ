// Package corretto implements the OctoJ provider for Amazon Corretto.
package corretto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
)

const (
	correttoBaseURL  = "https://corretto.aws/downloads/latest"
	providerName     = "corretto"
)

// supportedVersions lists the major versions supported by Amazon Corretto.
var supportedVersions = []string{"8", "11", "17", "21"}

// Provider implements the Amazon Corretto JDK provider.
type Provider struct {
	client *http.Client
}

// New creates a new Corretto provider.
func New() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// Search returns available Corretto JDK releases for the given version, OS and arch.
func (p *Provider) Search(ctx context.Context, version string, os string, arch string) ([]providers.JDKRelease, error) {
	if version == "" {
		// Return all supported versions
		var releases []providers.JDKRelease
		for _, v := range supportedVersions {
			release, err := p.buildRelease(ctx, v, os, arch)
			if err != nil {
				log.Debug().Err(err).Str("version", v).Msg("skipping version")
				continue
			}
			releases = append(releases, *release)
		}
		return releases, nil
	}

	// Check if version is supported
	if !isSupportedVersion(version) {
		return nil, fmt.Errorf("Corretto does not support Java %s — supported versions: %s",
			version, strings.Join(supportedVersions, ", "))
	}

	release, err := p.buildRelease(ctx, version, os, arch)
	if err != nil {
		return nil, err
	}

	return []providers.JDKRelease{*release}, nil
}

// GetRelease returns the best matching Corretto release for the given version, OS and arch.
func (p *Provider) GetRelease(ctx context.Context, version string, os string, arch string) (*providers.JDKRelease, error) {
	releases, err := p.Search(ctx, version, os, arch)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Corretto JDK %s release found for %s/%s", version, os, arch)
	}

	return &releases[0], nil
}

// buildRelease constructs a JDKRelease for the given version, OS and arch.
func (p *Provider) buildRelease(ctx context.Context, version, os, arch string) (*providers.JDKRelease, error) {
	det := &platform.Info{OS: os, Arch: arch}
	downloadURL := buildCorrettoURL(version, det)
	checksumURL := downloadURL + ".md5"

	log.Debug().
		Str("url", downloadURL).
		Str("checksum_url", checksumURL).
		Msg("checking Corretto availability")

	// Verify the release exists by checking the checksum URL
	checksum, err := p.fetchText(ctx, checksumURL)
	if err != nil {
		return nil, fmt.Errorf("Corretto JDK %s not available for %s/%s: %w", version, os, arch, err)
	}

	ext := det.ArchiveExt()
	fileName := buildCorrettoFileName(version, det)

	// Determine full version by following redirect or parsing filename
	fullVersion := fmt.Sprintf("%s.latest", version)

	// Try to get the actual version from the redirect
	actualURL, err := p.resolveRedirect(ctx, downloadURL)
	if err == nil {
		// Extract version from URL like: amazon-corretto-21.0.3.9.1-linux-x64.tar.gz
		fullVersion = extractVersionFromURL(actualURL, version)
	}

	return &providers.JDKRelease{
		Provider:     providerName,
		Version:      version,
		FullVersion:  fullVersion,
		OS:           os,
		Arch:         arch,
		URL:          downloadURL,
		Checksum:     strings.TrimSpace(checksum),
		ChecksumType: "md5",
		FileName:     fileName + ext,
		Size:         0, // Not available without downloading
	}, nil
}

// buildCorrettoURL constructs the Corretto download URL for a given version and platform.
func buildCorrettoURL(version string, det *platform.Info) string {
	archStr := det.CorrettoArch()
	osStr := det.CorrettoOS()

	ext := det.ArchiveExt()
	return fmt.Sprintf("%s/amazon-corretto-%s-%s-%s-jdk%s",
		correttoBaseURL, version, archStr, osStr, ext)
}

// buildCorrettoFileName returns the expected filename for a Corretto download.
func buildCorrettoFileName(version string, det *platform.Info) string {
	return fmt.Sprintf("amazon-corretto-%s-%s-%s-jdk",
		version, det.CorrettoArch(), det.CorrettoOS())
}

// fetchText fetches a text URL and returns its content.
func (p *Provider) fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// resolveRedirect follows redirects to get the final URL.
func (p *Provider) resolveRedirect(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")

	// Don't follow redirects automatically so we can see the final URL
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if loc := resp.Header.Get("Location"); loc != "" {
		return loc, nil
	}

	return resp.Request.URL.String(), nil
}

// extractVersionFromURL tries to parse the full version from the redirect URL.
// e.g., ".../amazon-corretto-21.0.3.9.1-linux-x64.tar.gz" → "21.0.3.9.1"
func extractVersionFromURL(fullURL, majorVersion string) string {
	parts := strings.Split(fullURL, "/")
	if len(parts) == 0 {
		return majorVersion + ".latest"
	}
	filename := parts[len(parts)-1]
	// filename like: amazon-corretto-21.0.3.9.1-linux-x64.tar.gz
	prefix := "amazon-corretto-"
	if strings.HasPrefix(filename, prefix) {
		rest := filename[len(prefix):]
		dashIdx := strings.Index(rest, "-")
		if dashIdx > 0 {
			return rest[:dashIdx]
		}
	}
	return majorVersion + ".latest"
}

// isSupportedVersion returns true if the version is in the supported list.
func isSupportedVersion(version string) bool {
	for _, v := range supportedVersions {
		if v == version {
			return true
		}
	}
	return false
}
