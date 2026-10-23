package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"io"
	"net/http"
	"os"
	"strings"
)

const maxChecksumMetadata = 1 << 20

func downloadFile(ctx context.Context, client sysutil.HTTPClient, url, destPath string, expectedSize int64) error {
	resp, err := getURL(ctx, client, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, copyErr := io.Copy(out, io.LimitReader(resp.Body, expectedSize+1))
	if copyErr != nil {
		return copyErr
	}
	if written != expectedSize {
		return fmt.Errorf("release archive size mismatch: expected %d, got %d", expectedSize, written)
	}
	err = out.Close()
	return err
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
