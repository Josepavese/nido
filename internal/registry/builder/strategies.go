package builder

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/image"
)

// Default client with timeout
var client = &http.Client{
	Timeout: 30 * time.Second,
}

// Fetch dispatches to the correct strategy implementation
func Fetch(src Source, strat Strategy) ([]image.Version, error) {
	switch strat.Type {
	case "generic":
		return fetchGeneric(src, strat)
	case "ubuntu-cloud":
		return fetchUbuntu(src, strat)
	case "debian-cloud":
		return fetchDebian(src, strat)
	case "alpine-cdn":
		return fetchAlpine(src, strat)
	default:
		return nil, fmt.Errorf("unknown strategy type: %s", strat.Type)
	}
}

// --- Strategy Implementations ---

func fetchGeneric(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// 1. Construct URLs
		replacer := strings.NewReplacer("{base_url}", strat.BaseURL, "{version}", ver)
		imgURL := replacer.Replace(strat.TemplateURL)
		checksumURL := replacer.Replace(strat.ChecksumURL)
		regex := replacer.Replace(strat.Regex)

		// 2. Verify Image Existence (HEAD)
		size, err := getRemoteSize(imgURL)
		if err != nil {
			fmt.Printf("⚠️ [%s] Image not found for %s: %v\n", src.Name, ver, err)
			continue
		}

		// 3. Fetch Checksum
		chk, err := findChecksum(checksumURL, regex, strat.ChecksumType)
		if err != nil {
			fmt.Printf("⚠️ [%s] Checksum failed for %s: %v\n", src.Name, ver, err)
			continue
		}

		results = append(results, image.Version{
			Version:      ver,
			Aliases:      []string{ver}, // TODO: Add aliases config if needed
			Arch:         "amd64",       // TODO: Make configurable
			URL:          imgURL,
			ChecksumType: strat.ChecksumType,
			Checksum:     chk,
			SizeBytes:    size,
			Format:       "qcow2", // Assumed for generic tier 1
		})
	}
	return results, nil
}

func fetchUbuntu(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// Ubuntu structure: {base}/{ver}/release/SHA256SUMS
		sumsURL := fmt.Sprintf("%s/%s/release/SHA256SUMS", strat.BaseURL, ver)

		// Ubuntu images usually match *server-cloudimg-amd64.img
		// We fetch the SUMS file first to get the exact filename and hash
		content, err := fetchString(sumsURL)
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to fetch SUMS for %s\n", src.Name, ver)
			continue
		}

		// Parse SUMS
		// Format: <hash> *<filename>
		lineRegex := regexp.MustCompile(`([a-f0-9]{64}) \*(.+server-cloudimg-amd64\.img)`)
		match := lineRegex.FindStringSubmatch(content)
		if match == nil {
			fmt.Printf("⚠️ [%s] No matching image found in SUMS for %s\n", src.Name, ver)
			continue
		}

		hash := match[1]
		filename := match[2]
		imgURL := fmt.Sprintf("%s/%s/release/%s", strat.BaseURL, ver, filename)

		// Get Size
		size, _ := getRemoteSize(imgURL)

		// Aliases (hardcoded logic for common ubuntu versions)
		aliases := []string{}
		if ver == "24.04" {
			aliases = []string{"noble", "lts", "latest"}
		}
		if ver == "22.04" {
			aliases = []string{"jammy"}
		}

		results = append(results, image.Version{
			Version:      ver,
			Aliases:      aliases,
			Arch:         "amd64",
			URL:          imgURL,
			ChecksumType: "sha256",
			Checksum:     hash,
			SizeBytes:    size,
			Format:       "qcow2",
		})
	}
	return results, nil
}

func fetchDebian(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// Debian: {base}/bookworm/latest/SHA512SUMS (we need codename mapping or config)
		codename := "bookworm" // default
		if ver == "11" {
			codename = "bullseye"
		}

		baseURL := fmt.Sprintf("%s/%s/latest", strat.BaseURL, codename)
		sumsURL := baseURL + "/SHA512SUMS"

		content, err := fetchString(sumsURL)
		if err != nil {
			continue
		}

		// debian-12-genericcloud-amd64.qcow2
		// Match hash, then spaces, then optional *, then filename
		lineRegex := regexp.MustCompile(`([a-f0-9]{128})\s+\*?((debian-\d+-genericcloud-amd64\.qcow2))`)
		match := lineRegex.FindStringSubmatch(content)
		if match == nil {
			continue
		}

		hash := match[1]
		filename := match[2] // Group 2 is filename

		if filename == "" {
			continue
		}

		imgURL := baseURL + "/" + filename
		size, _ := getRemoteSize(imgURL)

		results = append(results, image.Version{
			Version:      ver,
			Aliases:      []string{codename},
			Arch:         "amd64",
			URL:          imgURL,
			ChecksumType: "sha512",
			Checksum:     hash,
			SizeBytes:    size,
			Format:       "qcow2",
		})
	}
	return results, nil
}

func fetchAlpine(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// URL: https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/cloud/
		baseURL := fmt.Sprintf("%s/v%s/releases/cloud", strat.BaseURL, ver)

		// 1. Fetch directory listing
		content, err := fetchString(baseURL + "/")
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to list dir for %s: %v\n", src.Name, ver, err)
			continue
		}

		// 2. Find latest image
		// Pattern: nocloud_alpine-3.21.0-x86_64-bios-cloudinit-r0.qcow2
		// Regex: href="(nocloud_alpine-<ver>(\.\d+)*-x86_64-bios-cloudinit-r\d+\.qcow2)"
		regexStr := fmt.Sprintf(`href="(nocloud_alpine-%s\.\d+-x86_64-bios-cloudinit-r\d+\.qcow2)"`, regexp.QuoteMeta(ver))
		re := regexp.MustCompile(regexStr)

		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			fmt.Printf("⚠️ [%s] No image found for %s in %s\n", src.Name, ver, baseURL)
			continue
		}

		// Take the last one (assuming higher version/revision is later in list)
		filename := matches[len(matches)-1][1]
		imgURL := baseURL + "/" + filename
		checksumURL := imgURL + ".sha512"

		// 3. Fetch Checksum (it's usually just the hash in the file)
		hash, err := findChecksum(checksumURL, "", "sha512")
		if err != nil {
			fmt.Printf("⚠️ [%s] Checksum failed for %s: %v\n", src.Name, ver, err)
			continue
		}

		// 4. Get Size
		size, _ := getRemoteSize(imgURL)

		results = append(results, image.Version{
			Version:      ver,
			Aliases:      []string{ver, "latest"},
			Arch:         "amd64",
			URL:          imgURL,
			ChecksumType: "sha512",
			Checksum:     hash,
			SizeBytes:    size,
			Format:       "qcow2",
		})
	}
	return results, nil
}

// --- Helpers ---

func getRemoteSize(url string) (int64, error) {
	resp, err := client.Head(url)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	return resp.ContentLength, nil
}

func fetchString(url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	return string(bytes), err
}

func findChecksum(url, regexPattern, algo string) (string, error) {
	content, err := fetchString(url)
	if err != nil {
		return "", err
	}

	if regexPattern != "" {
		// Use regex to find line matching filename pattern, then extract hash
		// Standard format: HASH  FILENAME
		// We assume limits: SHA256 (64 chars), SHA512 (128 chars)

		scanner := bufio.NewScanner(strings.NewReader(content))
		re := regexp.MustCompile(regexPattern)

		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					return fields[0], nil // Hash is usually first
				}
			}
		}
		return "", fmt.Errorf("no match for regex %s", regexPattern)
	}

	// If no regex, maybe the file IS the checksum (e.g. .sha256 file with just hash)
	hash := strings.TrimSpace(content)
	// simple validation
	if algo == "sha256" && len(hash) == 64 {
		return hash, nil
	}

	// Try parsing first word
	fields := strings.Fields(hash)
	if len(fields) > 0 {
		return fields[0], nil
	}

	return "", fmt.Errorf("could not parse checksum")
}
