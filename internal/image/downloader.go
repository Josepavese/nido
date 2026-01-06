package image

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Downloader handles downloading files with resumption and progress tracking.
type Downloader struct {
	// Quiet suppresses progress output if true
	Quiet bool
}

// Download downloads a file from url to dest.
// It supports resumption if dest+".part" exists.
// expectedSize is optional (0 to ignore), used for progress calculation if Content-Length is missing.
func (d *Downloader) Download(url, dest string, expectedSize int64) error {
	partPath := dest + ".part"

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var startByte int64 = 0

	// Check for existing partial file
	if info, err := os.Stat(partPath); err == nil {
		startByte = info.Size()
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle status codes
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		// File likely already fully downloaded or server doesn't support range
		// For safety, let's restart if size mismatches, or assume done if matches?
		// Better strategy: if 416, delete part and retry without range
		os.Remove(partPath)
		return d.Download(url, dest, expectedSize)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Open partial file
	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		// Server didn't support range or we started fresh
		startByte = 0
		flags |= os.O_TRUNC
	}

	out, err := os.OpenFile(partPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer out.Close()

	// Setup progress tracking
	totalSize := resp.ContentLength + startByte
	if expectedSize > 0 && totalSize == 0 {
		totalSize = expectedSize
	}

	counter := &writeCounter{
		total:   uint64(totalSize),
		current: uint64(startByte),
		quiet:   d.Quiet,
	}

	// Copy data
	if _, err := io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Finish progress
	counter.Finish()

	// Close file before rename
	out.Close()

	// Verify size if known
	if totalSize > 0 {
		if info, err := os.Stat(partPath); err == nil {
			if info.Size() != int64(totalSize) {
				return fmt.Errorf("size mismatch: expected %d, got %d", totalSize, info.Size())
			}
		}
	}

	// Rename .part to final
	if err := os.Rename(partPath, dest); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	return nil
}

// DownloadMultiPart downloads multiple sequential parts and concatenates them into dest.
func (d *Downloader) DownloadMultiPart(urls []string, dest string, expectedTotalSize int64) error {
	tmpDir, err := os.MkdirTemp("", "nido-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var parts []string
	for i, url := range urls {
		partDest := filepath.Join(tmpDir, fmt.Sprintf("part.%03d", i+1))
		if !d.Quiet {
			fmt.Printf("🌐 Downloading part %d/%d...\n", i+1, len(urls))
		}
		if err := d.Download(url, partDest, 0); err != nil {
			return fmt.Errorf("failed to download part %d: %w", i+1, err)
		}
		parts = append(parts, partDest)
	}

	if !d.Quiet {
		fmt.Printf("🧩 Reassembling image...\n")
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create final image: %w", err)
	}
	defer out.Close()

	for _, part := range parts {
		f, err := os.Open(part)
		if err != nil {
			return fmt.Errorf("failed to open part %s: %w", part, err)
		}
		if _, err := io.Copy(out, f); err != nil {
			f.Close()
			return fmt.Errorf("failed to concatenate part %s: %w", part, err)
		}
		f.Close()
	}

	if expectedTotalSize > 0 {
		info, err := os.Stat(dest)
		if err == nil && info.Size() != expectedTotalSize {
			return fmt.Errorf("final size mismatch: expected %d, got %d", expectedTotalSize, info.Size())
		}
	}

	return nil
}

// Decompress extracts an archive to a destination.
// Currently supported: .tar.xz (standard for Kali cloud images)
func (d *Downloader) Decompress(src, dest string) error {
	if !d.Quiet {
		fmt.Printf("📦 Decompressing %s...\n", filepath.Base(src))
	}

	if strings.HasSuffix(src, ".tar.xz") {
		// Kali images are tarballs containing the qcow2
		// tar -xJf src -C dir
		destDir := filepath.Dir(dest)
		cmd := exec.Command("tar", "-xJf", src, "-C", destDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("extraction failed: %v (%s)", err, string(out))
		}

		// Kali convention: kali-linux-2025.4-genericcloud-amd64.tar.xz
		// contains 'disk.raw' (but in qcow2 format despite the extension!)
		// We try several candidates for what was extracted
		candidates := []string{
			strings.TrimSuffix(filepath.Base(src), ".tar.xz") + ".qcow2",
			"disk.raw",
		}

		for _, cand := range candidates {
			candPath := filepath.Join(destDir, cand)
			if _, err := os.Stat(candPath); err == nil {
				if candPath != dest {
					if err := os.Rename(candPath, dest); err != nil {
						return fmt.Errorf("failed to move extracted image (%s -> %s): %w", candPath, dest, err)
					}
				}
				return nil
			}
		}

		return fmt.Errorf("extraction succeeded but could not find extracted image in %s", destDir)
	}

	return fmt.Errorf("unsupported compression format for %s", src)
}

// writeCounter counts bytes written and prints progress
type writeCounter struct {
	total   uint64
	current uint64
	quiet   bool
	lastUpd time.Time
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.current += uint64(n)
	wc.Print()
	return n, nil
}

func (wc *writeCounter) Print() {
	if wc.quiet {
		return
	}

	// Update at most every 100ms prevents flickering
	if time.Since(wc.lastUpd) < 100*time.Millisecond && wc.current < wc.total {
		return
	}
	wc.lastUpd = time.Now()

	// Clear line
	fmt.Print("\r")

	if wc.total == 0 {
		// Unknown size
		fmt.Printf("📦 Downloading... %d MB", wc.current/1024/1024)
		return
	}

	percent := float64(wc.current) / float64(wc.total) * 100
	width := 40
	completed := int(percent / 100 * float64(width))

	bar := strings.Repeat("█", completed) + strings.Repeat("░", width-completed)
	fmt.Printf("📦 %s %.1f%% (%d/%d MB)", bar, percent, wc.current/1024/1024, wc.total/1024/1024)
}

func (wc *writeCounter) Finish() {
	if !wc.quiet {
		fmt.Println()
	}
}
