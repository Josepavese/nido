package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"io"
	"net/http"
	"os"
	"strings"
)

const maxChecksumMetadata = 1 << 20

func downloadFile(ctx context.Context, client sysutil.HTTPClient, url, destPath string, expectedSize int64) error {
	ctx, cancel := context.WithTimeout(ctx, updateDownloadTimeout)
	defer cancel()
	transfer := image.NewTransfer(image.TransferOptions{Context: ctx, Client: releaseDownloadClient{client, expectedSize}})
	var err error
	for attempt := 0; attempt < updateDownloadAttempts; attempt++ {
		err = transfer.Download(url, destPath, expectedSize)
		// Resume interrupted bodies through the existing transfer owner. Invalid
		// ranges, HTTP errors and integrity failures never trigger a retry.
		if err == nil || ctx.Err() != nil || !errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
	}
	if err != nil {
		return err
	}
	info, err := os.Stat(destPath)
	if err == nil && info.Size() != expectedSize {
		err = fmt.Errorf("release archive size mismatch: expected %d, got %d", expectedSize, info.Size())
	}
	return err
}

// Keep untrusted response bodies bounded even when upstream ignores the size
// announced in release metadata. Close still reaches the transport on all paths.
type releaseDownloadClient struct {
	sysutil.HTTPClient
	expectedSize int64
}

func (c releaseDownloadClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err == nil {
		resp.Body = struct {
			io.Reader
			io.Closer
		}{io.LimitReader(resp.Body, c.expectedSize+1), resp.Body}
	}
	return resp, err
}

func verifyReleaseAssetChecksum(ctx context.Context, client sysutil.HTTPClient, checksumURL, assetName, archivePath string) error {
	resp, err := getURL(ctx, client, checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected checksum HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumMetadata+1))
	if err != nil {
		return err
	}
	if len(body) > maxChecksumMetadata {
		return fmt.Errorf("release checksum metadata too large")
	}
	expected := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("%s not listed in SHA256SUMS", assetName)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))

	if actual != expected {
		return fmt.Errorf("%s checksum mismatch", assetName)
	}
	return nil
}
