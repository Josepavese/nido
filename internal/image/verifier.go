package image

import (
	"bufio"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var verifierClient = &http.Client{
	Timeout: 30 * time.Second,
}

// FetchChecksum fetches a remote checksum and parses it using the provided regex.
func FetchChecksum(url, regexPattern, algo string) (string, error) {
	resp, err := verifierClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch checksum from %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	content := string(body)

	if regexPattern != "" {
		scanner := bufio.NewScanner(strings.NewReader(content))
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}

		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					return fields[0], nil
				}
			}
		}
		return "", fmt.Errorf("checksum entry not found for pattern %s in %s", regexPattern, url)
	}

	// Fallback: assume the file contains just the hash
	hash := strings.TrimSpace(content)
	fields := strings.Fields(hash)
	if len(fields) > 0 {
		return fields[0], nil
	}

	return "", fmt.Errorf("failed to parse checksum from %s", url)
}

// VerifyChecksum checks if the file at path matches the expected checksum.
// Supported algorithms: "sha256", "sha512".
func VerifyChecksum(path, expected, algorithm string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer f.Close()

	var h hash.Hash
	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}
