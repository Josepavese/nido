package image

import (
	"context"
	"errors"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Transfer owns download, resumption and decompression. Consumers can inject a
// substitute without depending on the concrete HTTP/process adapter.
type Transfer interface {
	Prepare(Version, string) error
	Download(string, string, int64) error
	DownloadMultiPart([]string, string, int64) error
	Decompress(string, string) error
}
type TransferOptions struct {
	OnProgress func(current, total int64)
	Client     sysutil.HTTPClient
	Context    context.Context
}
type downloader struct {
	OnProgress func(current, total int64)
	client     sysutil.HTTPClient
	ctx        context.Context
}

func NewTransfer(options TransferOptions) Transfer {
	client := options.Client
	if client == nil {
		client = &http.Client{}
	}
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return &downloader{OnProgress: options.OnProgress, client: client, ctx: ctx}
}

// Download downloads a file from url to dest.
// It supports resumption if dest+".part" exists.
// expectedSize is optional (0 to ignore), used for progress calculation if Content-Length is missing.
func (d *downloader) Download(url, dest string, expectedSize int64) error {
	return d.download(url, dest, expectedSize, nil)
}
func (d *downloader) download(url, dest string, expectedSize int64, verify func(string) error) (retErr error) {
	lease, err := sysutil.AcquireResource(dest + ".download.lock")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lease.Close()) }()

	partPath := dest + ".part"

	// Create request
	req, err := http.NewRequestWithContext(d.ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var startByte int64 = 0

	// Check for existing partial file
	if info, err := os.Lstat(partPath); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("partial download must be a regular file")
		}
		startByte = info.Size()
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle status codes
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return fmt.Errorf("upstream rejected resume range at byte %d; partial download preserved", startByte)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	if resp.StatusCode == http.StatusPartialContent {
		var first, last, total int64
		if _, err := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &first, &last, &total); err != nil || first != startByte || last < first || total <= last || (resp.ContentLength >= 0 && last-first+1 != resp.ContentLength) {
			return fmt.Errorf("invalid resume Content-Range %q; partial download preserved", resp.Header.Get("Content-Range"))
		}
	}

	resume := resp.StatusCode == http.StatusPartialContent
	if !resume {
		startByte = 0
	}
	out, err := sysutil.OpenDownloadFile(partPath, resume, startByte)

	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer out.Close()

	// Setup progress tracking
	totalSize := expectedSize
	if resp.ContentLength >= 0 {
		totalSize = resp.ContentLength + startByte
	}
	if totalSize < 0 {
		totalSize = 0
	}

	counter := &writeCounter{
		total:      uint64(totalSize),
		current:    uint64(startByte),
		onProgress: d.OnProgress,
	}

	// Copy data
	if _, err := io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Finish progress
	counter.Finish()

	// Close file before rename
	if err := out.Close(); err != nil {
		return err
	}

	// Verify size if known
	if totalSize > 0 {
		if info, err := os.Stat(partPath); err == nil {
			if info.Size() != int64(totalSize) {
				return fmt.Errorf("size mismatch: expected %d, got %d", totalSize, info.Size())
			}
		}
	}

	if verify != nil {
		if err := verify(partPath); err != nil {
			return err
		}
	}

	// Rename .part to final
	if err := sysutil.PublishFile(partPath, dest); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	return nil
}

// DownloadMultiPart downloads multiple sequential parts and concatenates them into dest.
func (d *downloader) DownloadMultiPart(urls []string, dest string, expectedTotalSize int64) error {
	return d.downloadMultiPart(urls, dest, expectedTotalSize, nil)
}
func (d *downloader) downloadMultiPart(urls []string, dest string, expectedTotalSize int64, verify func(string) error) error {

	if len(urls) == 0 {
		return fmt.Errorf("multipart image has no parts")
	}
	stage, err := os.CreateTemp(filepath.Dir(dest), ".nido-multipart-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()
	chunk := stage.Name() + ".chunk"
	defer os.Remove(chunk)
	defer os.Remove(chunk + ".part")
	for i, url := range urls {
		if d.OnProgress != nil {
			d.OnProgress(int64(i), int64(len(urls)))
		}
		if err := d.Download(url, chunk, 0); err != nil {
			return fmt.Errorf("download part %d: %w", i+1, err)
		}
		part, err := os.Open(chunk)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(stage, part)
		closeErr := part.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if err := os.Remove(chunk); err != nil {
			return err
		}
	}
	info, err := stage.Stat()
	if err != nil {
		return err
	}
	if expectedTotalSize > 0 && info.Size() != expectedTotalSize {
		return fmt.Errorf("final size mismatch: expected %d, got %d", expectedTotalSize, info.Size())
	}
	if err := stage.Close(); err != nil {
		return err
	}
	if verify != nil {
		if err := verify(stage.Name()); err != nil {
			return err
		}
	}

	return sysutil.PublishFile(stage.Name(), dest)
}

// Prepare downloads, verifies, and decompresses a catalog version into dest.
// Locally supplied images may omit a checksum; any provided
// checksum is enforced before the image is made available.
func (d *downloader) Prepare(ver Version, dest string) error {
	downloadPath := dest
	isTarXz := strings.HasSuffix(ver.URL, ".tar.xz")
	isZst := strings.Contains(ver.URL, ".zst") || strings.Contains(ver.URL, ".zstandard")
	if len(ver.PartURLs) > 0 {
		first := ver.PartURLs[0]
		isTarXz = strings.HasSuffix(first, ".tar.xz")
		isZst = strings.Contains(first, ".zst") || strings.Contains(first, ".zstandard")
	}
	if isTarXz {
		downloadPath = dest + ".tar.xz"
	} else if isZst {
		downloadPath = dest + ".zst"
	}

	var verify func(string) error
	if ver.Checksum != "" {
		verify = func(path string) error { return VerifyChecksum(path, ver.Checksum, ver.ChecksumType) }
	}
	var err error
	if len(ver.PartURLs) > 0 {
		err = d.downloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes, verify)
	} else {
		err = d.download(ver.URL, downloadPath, ver.SizeBytes, verify)
	}
	if err != nil {
		return err
	}

	if isTarXz || isZst {
		if err := d.Decompress(downloadPath, dest); err != nil {
			_ = os.Remove(downloadPath)
			return err
		}
		_ = os.Remove(downloadPath)
	}

	return nil
}

// Decompress extracts an archive to a destination.
// Currently supported: .tar.xz (standard for Kali cloud images)
func (d *downloader) Decompress(src, dest string) error {
	stage, err := os.CreateTemp(filepath.Dir(dest), ".nido-decompress-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()
	copyOutput := func(reader io.Reader) error { _, err := io.Copy(stage, reader); return err }
	switch {
	case strings.HasSuffix(src, ".zst"), strings.HasSuffix(src, ".zstandard"):
		err = sysutil.ReadCommandOutput(d.ctx, "zstd", []string{"-d", "-c", src}, copyOutput)
	case strings.HasSuffix(src, ".tar.xz"):
		// Extract a single image to stdout. Archive paths and other members never get
		// filesystem write access to the user's image directory.
		candidateName := strings.TrimSuffix(filepath.Base(src), ".tar.xz") + ".qcow2"
		candidates := []string{candidateName, "./" + candidateName, "disk.raw", "./disk.raw"}
		for _, member := range candidates {
			if err = stage.Truncate(0); err != nil {
				return err
			}
			if _, err = stage.Seek(0, 0); err != nil {
				return err
			}
			err = sysutil.ReadCommandOutput(d.ctx, "tar", []string{"-xJOf", src, "--", member}, copyOutput)
			if err == nil {
				break
			}
		}
	default:
		return fmt.Errorf("unsupported compression format for %s", src)
	}
	if err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}
	info, err := stage.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("decompressed image is empty")
	}
	if err := stage.Close(); err != nil {
		return err
	}
	return sysutil.PublishFile(stage.Name(), dest)
}

type writeCounter struct {
	total      uint64
	current    uint64
	onProgress func(current, total int64)
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.current += uint64(n)
	wc.Print()
	return n, nil
}

func (wc *writeCounter) Print() {
	// Always call callback if present
	if wc.onProgress != nil {
		wc.onProgress(int64(wc.current), int64(wc.total))
	}

}

func (wc *writeCounter) Finish() {
	if wc.onProgress != nil {
		wc.onProgress(int64(wc.current), int64(wc.total))
	}
}
