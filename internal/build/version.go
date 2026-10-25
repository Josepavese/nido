package build

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Version is stamped by bin/build-release.sh through
// -ldflags "-X github.com/Josepavese/nido/v5/internal/build.Version=vX.Y.Z".
// It stays empty for source builds so init can recover the installed module
// version (`go install ...@latest`) instead of reporting a stale constant.
var Version string

func init() {
	if Version == "" {
		Version = sourceVersion()
	}
}

// sourceVersion returns the module version embedded by the Go toolchain.
func sourceVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := normalizeVersion(info.Main.Version); v != "" {
		return v
	}
	return "dev"
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "(devel)" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

const LatestReleaseURL = "https://api.github.com/repos/Josepavese/nido/releases/latest"
const maxReleaseMetadata = 1 << 20

type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}
type ReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// LatestRelease returns a single coherent release identity and asset set.
func LatestRelease(ctx context.Context, client sysutil.HTTPClient) (Release, error) {
	var release Release
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL, nil)
	if err != nil {
		return release, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return release, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release, fmt.Errorf("unexpected HTTP status from GitHub: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseMetadata+1))
	if err != nil {
		return release, err
	}
	if len(data) > maxReleaseMetadata {
		return release, fmt.Errorf("release metadata too large")
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return release, err
	}
	if release.TagName == "" {
		return release, fmt.Errorf("missing tag_name in GitHub response")
	}
	return release, nil
}
func GetLatestVersion() (string, error) {
	release, err := LatestRelease(context.Background(), &http.Client{Timeout: 3 * time.Second})
	return release.TagName, err
}
