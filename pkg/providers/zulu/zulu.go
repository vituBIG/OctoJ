// Package zulu implements the OctoJ provider for Azul Zulu JDK.
package zulu

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
	azulBaseURL  = "https://api.azul.com/metadata/v1/zulu/packages/"
	providerName = "zulu"
)

// Provider implements the Azul Zulu JDK provider.
type Provider struct {
	client *http.Client
}

// New creates a new Zulu provider.
func New() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// azulPackage represents a single package from the Azul Metadata API.
type azulPackage struct {
	PackageUUID       string `json:"package_uuid"`
	Name              string `json:"name"`
	JavaVersion       []int  `json:"java_version"`
	DownloadURL       string `json:"download_url"`
	SHA256HashValue   string `json:"sha256_hash_value"`
	LatestBuildNumber int    `json:"latest_build_number"`
	ProductVersion    string `json:"product_version"`
	Size              int64  `json:"size"`
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	JavaPackageType   string `json:"java_package_type"`
}

// Search returns available Zulu JDK releases for the given version, OS and arch.
func (p *Provider) Search(ctx context.Context, version string, os string, arch string) ([]providers.JDKRelease, error) {
	det := &platform.Info{OS: os, Arch: arch}

	params := url.Values{}
	if version != "" {
		params.Set("java_version", version)
	}
	params.Set("os", det.AzulOS())
	params.Set("arch", det.AzulArch())
	params.Set("hw_bitness", "64")
	params.Set("java_package_type", "jdk")
	params.Set("release_status", "ga")
	params.Set("availability_types", "CA")
	params.Set("page", "1")
	params.Set("page_size", "10")

	apiURL := azulBaseURL + "?" + params.Encode()
	log.Debug().Str("url", apiURL).Msg("calling Azul API")

	data, err := p.doRequest(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	return parseAzulPackages(data, os, arch)
}

// GetRelease returns the best matching Zulu release for the given version, OS and arch.
func (p *Provider) GetRelease(ctx context.Context, version string, os string, arch string) (*providers.JDKRelease, error) {
	major := platform.MajorVersion(version)
	releases, err := p.Search(ctx, major, os, arch)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no Zulu JDK %s release found for %s/%s", version, os, arch)
	}
	if version != major {
		for _, r := range releases {
			if r.FullVersion == version {
				return &r, nil
			}
		}
		return nil, fmt.Errorf("zulu JDK %s not found for %s/%s", version, os, arch)
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
		return nil, fmt.Errorf("azul API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

// parseAzulPackages converts Azul API JSON to JDKRelease slice.
func parseAzulPackages(data []byte, os, arch string) ([]providers.JDKRelease, error) {
	var packages []azulPackage
	if err := json.Unmarshal(data, &packages); err != nil {
		return nil, fmt.Errorf("failed to parse Azul API response: %w", err)
	}

	var releases []providers.JDKRelease
	for _, pkg := range packages {
		if pkg.DownloadURL == "" {
			continue
		}

		majorVersion := ""
		if len(pkg.JavaVersion) > 0 {
			majorVersion = fmt.Sprintf("%d", pkg.JavaVersion[0])
		}

		fullVersion := pkg.ProductVersion
		if fullVersion == "" && len(pkg.JavaVersion) > 0 {
			parts := make([]string, len(pkg.JavaVersion))
			for i, v := range pkg.JavaVersion {
				parts[i] = fmt.Sprintf("%d", v)
			}
			fullVersion = strings.Join(parts, ".")
		}
		if fullVersion == "" {
			fullVersion = majorVersion
		}

		releases = append(releases, providers.JDKRelease{
			Provider:     providerName,
			Version:      majorVersion,
			FullVersion:  fullVersion,
			OS:           os,
			Arch:         arch,
			URL:          pkg.DownloadURL,
			Checksum:     pkg.SHA256HashValue,
			ChecksumType: "sha256",
			FileName:     pkg.Name,
			Size:         pkg.Size,
		})
	}

	return releases, nil
}
