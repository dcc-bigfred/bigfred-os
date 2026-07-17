// Package update downloads hub binaries from GitHub Releases into /data/opt.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultInstallDir is the persistent override path for hub binaries.
	DefaultInstallDir = "/data/opt/bigfred/bin"

	DefaultBigFredRepo   = "dcc-bigfred/bigfred"
	DefaultBigFredOSRepo = "dcc-bigfred/bigfred-os"

	maxAssetBytes = 256 << 20 // 256 MiB
)

var (
	ErrUnknownTarget = errors.New("unknown update target")
	ErrNoRelease     = errors.New("no github release found")
	ErrNoAsset       = errors.New("no matching release asset")
)

// Target identifies which binary to refresh.
type Target string

const (
	TargetBigFred   Target = "bigfred"
	TargetBigFredUI Target = "bigfred-ui"
)

// Result is returned after a successful install.
type Result struct {
	Target  Target `json:"target"`
	Tag     string `json:"tag"`
	Asset   string `json:"asset"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Restart string `json:"restart"` // SysV service id to restart via Services
}

// Release is a GitHub release that contains the expected binary asset.
type Release struct {
	Tag         string `json:"tag"`
	Name        string `json:"name,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Prerelease  bool   `json:"prerelease"`
	Asset       string `json:"asset"`
}

// Config tunes GitHub download + install paths.
type Config struct {
	InstallDir    string
	BigFredRepo   string
	BigFredOSRepo string
	Arch          string // GOARCH; empty → runtime.GOARCH
	HTTPClient    *http.Client
	GitHubToken   string
}

// Updater installs release assets into InstallDir.
type Updater struct {
	cfg Config
}

func New(cfg Config) *Updater {
	if cfg.InstallDir == "" {
		cfg.InstallDir = DefaultInstallDir
	}
	if cfg.BigFredRepo == "" {
		cfg.BigFredRepo = DefaultBigFredRepo
	}
	if cfg.BigFredOSRepo == "" {
		cfg.BigFredOSRepo = DefaultBigFredOSRepo
	}
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Minute}
	}
	return &Updater{cfg: cfg}
}

type targetSpec struct {
	repo      string
	assetName string
	destName  string
	restart   string
}

func (u *Updater) spec(t Target) (targetSpec, error) {
	arch := assetArch(u.cfg.Arch)
	switch t {
	case TargetBigFred:
		return targetSpec{
			repo:      u.cfg.BigFredRepo,
			assetName: "loco-server-linux-" + arch,
			destName:  "bigfred",
			restart:   "bigfred",
		}, nil
	case TargetBigFredUI:
		return targetSpec{
			repo:      u.cfg.BigFredOSRepo,
			assetName: "bigfred-os-ui-linux-" + arch,
			destName:  "bigfred-os-ui",
			restart:   "bigfred-os-ui",
		}, nil
	default:
		return targetSpec{}, ErrUnknownTarget
	}
}

// ParseTarget maps API path segments onto Target values.
func ParseTarget(s string) (Target, error) {
	switch strings.TrimSpace(s) {
	case string(TargetBigFred), "loco-server":
		return TargetBigFred, nil
	case string(TargetBigFredUI), "bigfred-os-ui", "ui":
		return TargetBigFredUI, nil
	default:
		return "", ErrUnknownTarget
	}
}

const maxListedReleases = 30

// ListReleases returns recent GitHub releases that include the target asset.
func (u *Updater) ListReleases(ctx context.Context, t Target) ([]Release, error) {
	spec, err := u.spec(t)
	if err != nil {
		return nil, err
	}
	rels, err := u.listReleases(ctx, spec.repo)
	if err != nil {
		return nil, err
	}

	out := make([]Release, 0, len(rels))
	for _, rel := range rels {
		if rel.Draft {
			continue
		}
		asset, ok := findAsset(rel.Assets, spec.assetName)
		if !ok {
			continue
		}
		out = append(out, Release{
			Tag:         rel.TagName,
			Name:        rel.Name,
			PublishedAt: rel.PublishedAt,
			Prerelease:  rel.Prerelease,
			Asset:       asset.Name,
		})
	}
	return out, nil
}

// Apply downloads the matching asset for tag (empty = latest non-draft with asset)
// and atomically installs it.
func (u *Updater) Apply(ctx context.Context, t Target, tag string) (*Result, error) {
	spec, err := u.spec(t)
	if err != nil {
		return nil, err
	}

	tag = strings.TrimSpace(tag)
	var rel *ghRelease
	if tag == "" || tag == "latest" {
		rel, err = u.latestRelease(ctx, spec.repo)
	} else {
		rel, err = u.releaseByTag(ctx, spec.repo, tag)
	}
	if err != nil {
		return nil, err
	}

	asset, ok := findAsset(rel.Assets, spec.assetName)
	if !ok {
		return nil, fmt.Errorf("%w: want %q in %s %s", ErrNoAsset, spec.assetName, spec.repo, rel.TagName)
	}

	if err := os.MkdirAll(u.cfg.InstallDir, 0o755); err != nil {
		return nil, err
	}

	dest := filepath.Join(u.cfg.InstallDir, spec.destName)
	size, err := u.downloadInstall(ctx, asset.BrowserDownloadURL, dest)
	if err != nil {
		return nil, err
	}

	return &Result{
		Target:  t,
		Tag:     rel.TagName,
		Asset:   asset.Name,
		Path:    dest,
		Size:    size,
		Restart: spec.restart,
	}, nil
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt string    `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (u *Updater) latestRelease(ctx context.Context, repo string) (*ghRelease, error) {
	return u.fetchRelease(ctx, "https://api.github.com/repos/"+repo+"/releases/latest", repo)
}

func (u *Updater) releaseByTag(ctx context.Context, repo, tag string) (*ghRelease, error) {
	url := "https://api.github.com/repos/" + repo + "/releases/tags/" + tag
	return u.fetchRelease(ctx, url, repo+"@"+tag)
}

func (u *Updater) listReleases(ctx context.Context, repo string) ([]ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", repo, maxListedReleases)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	u.setGitHubHeaders(req)

	res, err := u.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrNoRelease, repo)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var rels []ghRelease
	if err := json.Unmarshal(body, &rels); err != nil {
		return nil, err
	}
	return rels, nil
}

func (u *Updater) fetchRelease(ctx context.Context, url, label string) (*ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	u.setGitHubHeaders(req)

	res, err := u.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrNoRelease, label)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("%w: %s", ErrNoRelease, label)
	}
	return &rel, nil
}

func findAsset(assets []ghAsset, name string) (ghAsset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return ghAsset{}, false
}

func (u *Updater) downloadInstall(ctx context.Context, url, dest string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	u.setGitHubHeaders(req)
	// GitHub asset CDN accepts the API accept header on browser_download_url too.
	req.Header.Set("Accept", "application/octet-stream")

	res, err := u.cfg.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2<<10))
		return 0, fmt.Errorf("download %s: %s: %s", url, res.Status, strings.TrimSpace(string(body)))
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".bigfred-update-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()

	n, err := io.Copy(tmp, io.LimitReader(res.Body, maxAssetBytes+1))
	if err != nil {
		return 0, err
	}
	if n > maxAssetBytes {
		return 0, fmt.Errorf("asset too large (>%d bytes)", maxAssetBytes)
	}
	if err := tmp.Chmod(0o755); err != nil {
		return 0, err
	}
	if err := tmp.Sync(); err != nil {
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return 0, err
	}
	keep = true
	return n, nil
}

func (u *Updater) setGitHubHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "bigfred-os-ui")
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := strings.TrimSpace(u.cfg.GitHubToken); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func assetArch(goarch string) string {
	switch goarch {
	case "arm":
		return "armv7"
	default:
		return goarch
	}
}
