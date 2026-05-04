// Package liberica implements the OctoJ provider for BellSoft Liberica JDK.
package liberica

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
)

const (
	bellsoftBaseURL = "https://api.bell-sw.com/v1/liberica/releases"
	providerName    = "liberica"
)

// Provider implements the BellSoft Liberica JDK provider.
type Provider struct {
	client *http.Client
}

// New creates a new Liberica provider.
func New() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// libericaRelease represents a single release from the BellSoft API.
type libericaRelease struct {
	Version        string `json:"version"`
	FeatureVersion int    `json:"featureVersion"`
	InterimVersion int    `json:"interimVersion"`
	UpdateVersion  int    `json:"updateVersion"`
	PatchVersion   int    `json:"patchVersion"`
	BuildVersion   int    `json:"buildVersion"`
	DownloadURL    string `json:"downloadUrl"`
	SHA1           string `json:"sha1"`
	SHA256         string `json:"sha256"`
	Filename       string `json:"filename"`
	Size           int64  `json:"size"`
	OS             string `json:"os"`
	Arch           string `json:"architecture"`
	BundleType     string `json:"bundleType"`
	ReleaseType    string `json:"releaseType"`
}

// Search returns available Liberica JDK releases for the given version, OS and arch.
func (p *Provider) Search(ctx context.Context, version string, osName string, arch string) ([]providers.JDKRelease, error) {
	det := &platform.Info{OS: osName, Arch: arch}

	params := url.Values{}
	if version != "" {
		params.Set("version-feature", version)
	}
	params.Set("os", det.BellSoftOS())
	params.Set("arch", det.BellSoftArch())
	params.Set("bitness", "64")
	params.Set("bundle-type", "jdk")

	apiURL := bellsoftBaseURL + "?" + params.Encode()
	log.Debug().Str("url", apiURL).Msg("calling BellSoft API")

	data, err := p.doRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	return parseLibericaReleases(data, osName, arch)
}

// GetRelease returns the best matching Liberica release for the given version, OS and arch.
func (p *Provider) GetRelease(ctx context.Context, version string, osName string, arch string) (*providers.JDKRelease, error) {
	releases, err := p.Search(ctx, version, osName, arch)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Liberica JDK %s release found for %s/%s", version, osName, arch)
	}

	return &releases[0], nil
}

// doRequest performs an HTTP GET and returns the body bytes.
func (p *Provider) doRequest(ctx context.Context, apiURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BellSoft API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

// parseLibericaReleases converts BellSoft API JSON to JDKRelease slice.
func parseLibericaReleases(data []byte, osName, arch string) ([]providers.JDKRelease, error) {
	var releases []libericaRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse BellSoft API response: %w", err)
	}

	var result []providers.JDKRelease
	for _, r := range releases {
		if r.DownloadURL == "" {
			continue
		}

		// Prefer SHA256, fall back to SHA1
		checksum := r.SHA256
		checksumType := "sha256"
		if checksum == "" {
			checksum = r.SHA1
			checksumType = "sha1"
		}

		majorVersion := fmt.Sprintf("%d", r.FeatureVersion)

		result = append(result, providers.JDKRelease{
			Provider:     providerName,
			Version:      majorVersion,
			FullVersion:  r.Version,
			OS:           osName,
			Arch:         arch,
			URL:          r.DownloadURL,
			Checksum:     checksum,
			ChecksumType: checksumType,
			FileName:     r.Filename,
			Size:         r.Size,
		})
	}

	return result, nil
}
