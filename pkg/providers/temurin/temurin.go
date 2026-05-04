// Package temurin implements the OctoJ provider for Eclipse Temurin (Adoptium).
package temurin

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
	adoptiumBaseURL = "https://api.adoptium.net/v3"
	providerName    = "temurin"
)

// Provider implements the Temurin (Eclipse Adoptium) JDK provider.
type Provider struct {
	client *http.Client
}

// New creates a new Temurin provider.
func New() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// adoptiumRelease represents the Adoptium API release JSON structure.
type adoptiumRelease struct {
	ReleaseName string `json:"release_name"`
	Binary      struct {
		Package struct {
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
			Name     string `json:"name"`
			Size     int64  `json:"size"`
		} `json:"package"`
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
	} `json:"binary"`
	VersionData struct {
		Major    int    `json:"major"`
		Semver   string `json:"semver"`
		Optional string `json:"optional"`
	} `json:"version_data"`
}

// Search returns available Temurin JDK releases for the given version, OS and arch.
func (p *Provider) Search(ctx context.Context, version string, os string, arch string) ([]providers.JDKRelease, error) {
	det := &platform.Info{OS: os, Arch: arch}

	apiURL := fmt.Sprintf("%s/assets/feature_releases/%s/ga", adoptiumBaseURL, version)

	params := url.Values{}
	params.Set("os", det.AdoptiumOS())
	params.Set("architecture", det.AdoptiumArch())
	params.Set("image_type", "jdk")
	params.Set("jvm_impl", "hotspot")
	params.Set("page", "0")
	params.Set("page_size", "5")

	fullURL := apiURL + "?" + params.Encode()

	log.Debug().Str("url", fullURL).Msg("calling Adoptium API")

	resp, err := p.doRequest(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	return parseAdoptiumReleases(resp, os, arch)
}

// GetRelease returns the best matching Temurin release for the given version, OS and arch.
func (p *Provider) GetRelease(ctx context.Context, version string, os string, arch string) (*providers.JDKRelease, error) {
	releases, err := p.Search(ctx, version, os, arch)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no Temurin JDK %s release found for %s/%s", version, os, arch)
	}

	return &releases[0], nil
}

// doRequest performs an HTTP GET and returns the response body bytes.
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found (HTTP 404)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Adoptium API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

// parseAdoptiumReleases converts the Adoptium API JSON response to JDKRelease slice.
func parseAdoptiumReleases(data []byte, os, arch string) ([]providers.JDKRelease, error) {
	var raw []adoptiumRelease
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Adoptium response: %w", err)
	}

	if len(raw) == 0 {
		log.Warn().Str("os", os).Str("arch", arch).Msg("Adoptium API returned 0 releases")
	}

	var releases []providers.JDKRelease
	for _, r := range raw {
		if r.Binary.Package.Link == "" {
			continue
		}

		releases = append(releases, providers.JDKRelease{
			Provider:     providerName,
			Version:      fmt.Sprintf("%d", r.VersionData.Major),
			FullVersion:  r.VersionData.Semver,
			OS:           os,
			Arch:         arch,
			URL:          r.Binary.Package.Link,
			Checksum:     r.Binary.Package.Checksum,
			ChecksumType: "sha256",
			FileName:     r.Binary.Package.Name,
			Size:         r.Binary.Package.Size,
		})
	}

	return releases, nil
}
