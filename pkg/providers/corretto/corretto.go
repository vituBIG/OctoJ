// Package corretto implements the OctoJ provider for Amazon Corretto.
package corretto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/OctavoBit/octoj/internal/platform"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
)

const (
	correttoBaseURL = "https://corretto.aws/downloads/latest"
	githubAPIBase   = "https://api.github.com/repos/corretto/corretto-%s/releases"
	providerName    = "corretto"
)

// supportedVersions lists the major versions supported by Amazon Corretto.
var supportedVersions = []string{"8", "11", "17", "21"}

// Provider implements the Amazon Corretto JDK provider.
type Provider struct {
	client *http.Client
}

// New creates a new Corretto provider.
func New() *Provider {
	return &Provider{client: &http.Client{}}
}

// Name returns the provider name.
func (p *Provider) Name() string { return providerName }

// githubRelease is a single GitHub release entry.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset is a downloadable file attached to a GitHub release.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Search returns available Corretto JDK releases for the given version, OS and arch.
func (p *Provider) Search(ctx context.Context, version string, os string, arch string) ([]providers.JDKRelease, error) {
	if version == "" {
		var releases []providers.JDKRelease
		for _, v := range supportedVersions {
			rels, err := p.searchVersion(ctx, v, os, arch)
			if err != nil {
				log.Debug().Err(err).Str("version", v).Msg("skipping corretto version")
				continue
			}
			releases = append(releases, rels...)
		}
		return releases, nil
	}

	if !isSupportedVersion(version) {
		return nil, fmt.Errorf("corretto does not support Java %s — supported versions: %s",
			version, strings.Join(supportedVersions, ", "))
	}

	return p.searchVersion(ctx, version, os, arch)
}

// GetRelease returns the best matching Corretto release for the given version, OS and arch.
func (p *Provider) GetRelease(ctx context.Context, version string, os string, arch string) (*providers.JDKRelease, error) {
	major := platform.MajorVersion(version)
	releases, err := p.searchVersion(ctx, major, os, arch)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no Corretto JDK %s release found for %s/%s", version, os, arch)
	}
	if version != major {
		for _, r := range releases {
			if r.FullVersion == version {
				return &r, nil
			}
		}
		return nil, fmt.Errorf("corretto JDK %s not found for %s/%s", version, os, arch)
	}
	return &releases[0], nil
}

// searchVersion queries GitHub releases for one major version and returns matching platform assets.
func (p *Provider) searchVersion(ctx context.Context, version, osName, arch string) ([]providers.JDKRelease, error) {
	det := &platform.Info{OS: osName, Arch: arch}
	suffix := fmt.Sprintf("-%s-%s-jdk%s", det.CorrettoOS(), det.CorrettoArch(), det.ArchiveExt())

	ghReleases, err := p.fetchGitHubReleases(ctx, version)
	if err != nil {
		log.Debug().Err(err).Str("version", version).Msg("GitHub API failed, falling back to latest URL")
		rel, err2 := p.buildLatestRelease(ctx, version, osName, arch)
		if err2 != nil {
			return nil, err2
		}
		return []providers.JDKRelease{*rel}, nil
	}

	var releases []providers.JDKRelease
	for _, r := range ghReleases {
		for _, asset := range r.Assets {
			if strings.HasSuffix(asset.Name, suffix) {
				releases = append(releases, providers.JDKRelease{
					Provider:    providerName,
					Version:     version,
					FullVersion: r.TagName,
					OS:          osName,
					Arch:        arch,
					URL:         asset.BrowserDownloadURL,
					FileName:    asset.Name,
					Size:        asset.Size,
				})
				break
			}
		}
	}

	if len(releases) == 0 {
		rel, err := p.buildLatestRelease(ctx, version, osName, arch)
		if err != nil {
			return nil, err
		}
		return []providers.JDKRelease{*rel}, nil
	}

	return releases, nil
}

// fetchGitHubReleases queries the GitHub releases API for a given Corretto major version.
func (p *Provider) fetchGitHubReleases(ctx context.Context, version string) ([]githubRelease, error) {
	apiURL := fmt.Sprintf(githubAPIBase+"?per_page=5", version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned HTTP %d: %s", resp.StatusCode, body)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}
	return releases, nil
}

// buildLatestRelease constructs a JDKRelease using the Corretto "latest" redirect URL.
func (p *Provider) buildLatestRelease(ctx context.Context, version, os, arch string) (*providers.JDKRelease, error) {
	det := &platform.Info{OS: os, Arch: arch}
	downloadURL := buildCorrettoURL(version, det)

	log.Debug().Str("url", downloadURL).Msg("checking Corretto latest availability")

	if err := p.checkExists(ctx, downloadURL); err != nil {
		return nil, fmt.Errorf("corretto JDK %s not available for %s/%s: %w", version, os, arch, err)
	}

	ext := det.ArchiveExt()
	fileName := buildCorrettoFileName(version, det)
	checksum, _ := p.fetchText(ctx, downloadURL+".md5")

	fullVersion := fmt.Sprintf("%s.latest", version)
	if actualURL, err := p.resolveRedirect(ctx, downloadURL); err == nil {
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
	}, nil
}

func buildCorrettoURL(version string, det *platform.Info) string {
	return fmt.Sprintf("%s/amazon-corretto-%s-%s-%s-jdk%s",
		correttoBaseURL, version, det.CorrettoArch(), det.CorrettoOS(), det.ArchiveExt())
}

func buildCorrettoFileName(version string, det *platform.Info) string {
	return fmt.Sprintf("amazon-corretto-%s-%s-%s-jdk", version, det.CorrettoArch(), det.CorrettoOS())
}

func (p *Provider) checkExists(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

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

func (p *Provider) resolveRedirect(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "octoj/0.1.0 (https://github.com/vituBIG/OctoJ)")
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

func extractVersionFromURL(fullURL, majorVersion string) string {
	parts := strings.Split(fullURL, "/")
	if len(parts) == 0 {
		return majorVersion + ".latest"
	}
	filename := parts[len(parts)-1]
	prefix := "amazon-corretto-"
	if strings.HasPrefix(filename, prefix) {
		rest := filename[len(prefix):]
		if idx := strings.Index(rest, "-"); idx > 0 {
			return rest[:idx]
		}
	}
	return majorVersion + ".latest"
}

func isSupportedVersion(version string) bool {
	for _, v := range supportedVersions {
		if v == version {
			return true
		}
	}
	return false
}
