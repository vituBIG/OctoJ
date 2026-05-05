// Package temurin implements the OctoJ provider for Eclipse Temurin (Adoptium).
package temurin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
)

// isMajorOnly returns true if version has no dots or plus signs (e.g. "17", not "17.0.18+8").
func isMajorOnly(version string) bool {
	for _, c := range version {
		if c == '.' || c == '+' {
			return false
		}
	}
	return true
}

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
// feature_releases returns a "binaries" array; assets/latest returns "binary" — handle both.
type adoptiumRelease struct {
	ReleaseName string `json:"release_name"`
	Binaries []struct {
		Package struct {
			Link         string `json:"link"`
			Checksum     string `json:"checksum"`
			ChecksumLink string `json:"checksum_link"`
			Name         string `json:"name"`
			Size         int64  `json:"size"`
		} `json:"package"`
	} `json:"binaries"`
	Binary struct {
		Package struct {
			Link         string `json:"link"`
			Checksum     string `json:"checksum"`
			ChecksumLink string `json:"checksum_link"`
			Name         string `json:"name"`
			Size         int64  `json:"size"`
		} `json:"package"`
	} `json:"binary"`
	VersionData struct {
		Major  int    `json:"major"`
		Semver string `json:"semver"`
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
	if !isMajorOnly(version) {
		// Full version like "17.0.18+8" — use the release_name endpoint directly.
		return p.getByReleaseName(ctx, "jdk-"+version, os, arch)
	}

	releases, err := p.Search(ctx, version, os, arch)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no Temurin JDK %s release found for %s/%s", version, os, arch)
	}
	return &releases[0], nil
}

// getByReleaseName finds a specific Temurin release by paginating feature_releases.
// The release_name API endpoint returns 404 even for valid releases, so we use
// feature_releases and match by full version string instead.
func (p *Provider) getByReleaseName(ctx context.Context, releaseName, osName, arch string) (*providers.JDKRelease, error) {
	wantVersion := releaseName[len("jdk-"):]

	var major int
	_, _ = fmt.Sscanf(releaseName, "jdk-%d", &major)
	if major == 0 {
		return nil, fmt.Errorf("cannot parse major version from release name %q", releaseName)
	}

	det := &platform.Info{OS: osName, Arch: arch}
	apiURL := fmt.Sprintf("%s/assets/feature_releases/%d/ga", adoptiumBaseURL, major)

	params := url.Values{}
	params.Set("os", det.AdoptiumOS())
	params.Set("architecture", det.AdoptiumArch())
	params.Set("image_type", "jdk")
	params.Set("jvm_impl", "hotspot")
	params.Set("page_size", "20")

	for page := 0; ; page++ {
		params.Set("page", fmt.Sprintf("%d", page))
		fullURL := apiURL + "?" + params.Encode()
		log.Debug().Str("url", fullURL).Msg("searching Adoptium feature_releases for specific version")

		data, err := p.doRequest(ctx, fullURL)
		if err != nil {
			return nil, err
		}
		releases, err := parseAdoptiumReleases(data, osName, arch)
		if err != nil {
			return nil, err
		}
		for i := range releases {
			if releases[i].FullVersion == wantVersion {
				return &releases[i], nil
			}
		}
		if len(releases) < 20 {
			break
		}
	}

	return nil, fmt.Errorf("temurin release %s not found for %s/%s", releaseName, osName, arch)
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
		return nil, fmt.Errorf("adoptium API returned HTTP %d: %s", resp.StatusCode, body)
	}

	return io.ReadAll(resp.Body)
}

// parseAdoptiumReleases converts the Adoptium API JSON response to JDKRelease slice.
// The feature_releases endpoint returns a JSON array; release_name returns a single object.
func parseAdoptiumReleases(data []byte, os, arch string) ([]providers.JDKRelease, error) {
	var raw []adoptiumRelease
	if err := json.Unmarshal(data, &raw); err != nil {
		// release_name endpoint returns a single object, not an array.
		var single adoptiumRelease
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse Adoptium response: %w", err)
		}
		raw = []adoptiumRelease{single}
	}

	if len(raw) == 0 {
		log.Warn().Str("os", os).Str("arch", arch).Msg("Adoptium API returned 0 releases")
	}

	var releases []providers.JDKRelease
	for _, r := range raw {
		// feature_releases uses "binaries" (array); assets/latest uses "binary" (object)
		pkg := r.Binary.Package
		for _, b := range r.Binaries {
			if b.Package.Link != "" {
				pkg = b.Package
				break
			}
		}
		if pkg.Link == "" {
			continue
		}

		major := r.VersionData.Major
		semver := r.VersionData.Semver

		// assets/latest omits version_data — parse from release_name e.g. "jdk-21.0.3+9"
		if major == 0 && r.ReleaseName != "" {
			_, _ = fmt.Sscanf(r.ReleaseName, "jdk-%d", &major)
			if major > 0 && len(r.ReleaseName) > 4 {
				semver = r.ReleaseName[4:]
			}
		}

		releases = append(releases, providers.JDKRelease{
			Provider:     providerName,
			Version:      fmt.Sprintf("%d", major),
			FullVersion:  semver,
			OS:           os,
			Arch:         arch,
			URL:          pkg.Link,
			Checksum:     pkg.Checksum,
			ChecksumLink: pkg.ChecksumLink,
			ChecksumType: "sha256",
			FileName:     pkg.Name,
			Size:         pkg.Size,
		})
	}

	return releases, nil
}
