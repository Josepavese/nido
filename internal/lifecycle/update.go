package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/internal/build"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
)

type UpdateRequest struct {
	CurrentVersion string
	NidoDir        string
	// Executable is an explicit target; empty selects the running executable.
	Executable string
}
type UpdateResult struct {
	Version        string
	AlreadyCurrent bool
	PreviousBinary string
	RegistryBackup string
	// RegistryError reports a partial result after the binary was installed.
	RegistryError error
}
type Updater interface {
	Update(context.Context, UpdateRequest) (UpdateResult, error)
}
type nativeUpdater struct{ client sysutil.HTTPClient }

func NewUpdater(client sysutil.HTTPClient) Updater { return &nativeUpdater{client: client} }

func getURL(ctx context.Context, client sysutil.HTTPClient, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (u *nativeUpdater) Update(ctx context.Context, request UpdateRequest) (UpdateResult, error) {
	var result UpdateResult
	if request.NidoDir == "" {
		return result, fmt.Errorf("nido data directory is required")
	}
	release, err := build.LatestRelease(ctx, u.client)
	if err != nil {
		return result, err
	}
	result.Version = release.TagName
	if release.TagName == request.CurrentVersion {
		result.AlreadyCurrent = true
		return result, nil
	}
	assetName, binaryName := build.NativeReleaseAsset()
	var assetURL, checksumURL string
	var assetSize int64
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			assetURL = asset.URL
			assetSize = asset.Size
		case "SHA256SUMS":
			checksumURL = asset.URL
		}
	}
	if assetURL == "" {
		return result, fmt.Errorf("release package %s is unavailable", assetName)
	}
	if assetSize <= 0 {
		return result, fmt.Errorf("release archive size is missing or invalid")
	}
	if checksumURL == "" {
		return result, fmt.Errorf("release is missing required SHA256SUMS")
	}
	target := request.Executable
	if target == "" {
		target, err = os.Executable()
		if err != nil {
			return result, err
		}
	}
	// Resolve the actual installation filesystem, including a Data-backed symlink.
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return result, err
	}
	backup := target + ".bak"
	if _, err := os.Lstat(backup); err == nil {
		return result, fmt.Errorf("previous binary %s is still present; retire it only after its consumers close", backup)
	} else if !os.IsNotExist(err) {
		return result, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(target), ".nido-update-*")
	if err != nil {
		return result, err
	}
	// Returning errors (rather than exiting) guarantees this cleanup on every path.
	defer os.RemoveAll(stage)
	archive := filepath.Join(stage, assetName)
	if err := downloadFile(ctx, u.client, assetURL, archive, assetSize); err != nil {
		return result, err
	}
	if err := verifyReleaseAssetChecksum(ctx, u.client, checksumURL, assetName, archive); err != nil {
		return result, err
	}
	candidate := filepath.Join(stage, binaryName)
	if err := extractBinaryFromReleaseAsset(archive, candidate, binaryName); err != nil {
		return result, err
	}
	if err := sysutil.MakeExecutable(candidate); err != nil {
		return result, err
	}
	if err := os.Rename(target, backup); err != nil {
		return result, fmt.Errorf("preserve previous binary: %w", err)
	}
	if err := os.Rename(candidate, target); err != nil {
		return result, errors.Join(fmt.Errorf("install updated binary: %w", err), os.Rename(backup, target))
	}
	// The previous executable can still be mapped by this process or MCP sessions.
	// Keep its pathname until those consumers are drained; never overwrite a backup.
	result.PreviousBinary = backup
	result.RegistryBackup, result.RegistryError = SyncBundledRegistryFromReleaseAsset(archive, request.NidoDir)
	return result, nil
}
