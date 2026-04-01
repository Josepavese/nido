package builder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
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
	case "opensuse-tumbleweed":
		return fetchOpenSUSE(src, strat)
	case "github-release":
		return fetchGithubRelease(src, strat)
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

		format := strat.Format
		if format == "" {
			format = "qcow2"
		}

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
			ChecksumURL:  checksumURL,
			ChecksumRegex: regex,
			SizeBytes:    size,
			Format:       format,
		})
	}
	return results, nil
}

func fetchUbuntu(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// 1. Fetch directory listing to find latest release folder
		baseURL := fmt.Sprintf("%s/%s", strat.BaseURL, ver)
		content, err := fetchString(baseURL + "/")
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to list dir for %s: %v\n", src.Name, ver, err)
			continue
		}

		// Find latest release folder: href="release-YYYYMMDD/"
		re := regexp.MustCompile(`href="(release-\d{8}/)"`)
		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			fmt.Printf("⚠️ [%s] No release folder found for %s\n", src.Name, ver)
			continue
		}

		// Extract folder names and sort alphabetically (YYYYMMDD sorts correctly)
		var folders []string
		for _, m := range matches {
			folders = append(folders, m[1])
		}
		sort.Strings(folders)
		latestReleaseFolder := strings.TrimSuffix(folders[len(folders)-1], "/")

		// 2. Fetch SUMS from the specific immutable folder
		sumsURL := fmt.Sprintf("%s/%s/%s/SHA256SUMS", strat.BaseURL, ver, latestReleaseFolder)
		sumsContent, err := fetchString(sumsURL)
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to fetch SUMS for %s %s\n", src.Name, ver, latestReleaseFolder)
			continue
		}

		// Parse SUMS. Format: <hash> *<filename>
		lineRegex := regexp.MustCompile(`([a-f0-9]{64}) \*(.+server-cloudimg-amd64\.img)`)
		sumsMatch := lineRegex.FindStringSubmatch(sumsContent)
		if sumsMatch == nil {
			fmt.Printf("⚠️ [%s] No matching image found in SUMS for %s %s\n", src.Name, ver, latestReleaseFolder)
			continue
		}

		hash := sumsMatch[1]
		filename := sumsMatch[2]
		imgURL := fmt.Sprintf("%s/%s/%s/%s", strat.BaseURL, ver, latestReleaseFolder, filename)

		// 3. Get Size
		size, _ := getRemoteSize(imgURL)

		// 4. Aliases
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
			ChecksumURL:  fmt.Sprintf("%s/%s/release/SHA256SUMS", strat.BaseURL, ver),
			ChecksumRegex: regexp.QuoteMeta(filename),
			SizeBytes:    size,
			Format:       "qcow2",
		})
	}
	return results, nil
}

func fetchDebian(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		codename := "bookworm" // default
		if ver == "11" {
			codename = "bullseye"
		}

		// 1. Fetch directory listing to find latest release folder
		baseURL := fmt.Sprintf("%s/%s", strat.BaseURL, codename)
		content, err := fetchString(baseURL + "/")
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to list dir for %s: %v\n", src.Name, ver, err)
			continue
		}

		// Find latest release folder: href="20260225-2399/"
		re := regexp.MustCompile(`href="(\d{8}-\d{4}/)"`)
		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			fmt.Printf("⚠️ [%s] No release folder found for %s\n", src.Name, ver)
			continue
		}

		// Extract folder names and sort
		var folders []string
		for _, m := range matches {
			folders = append(folders, m[1])
		}
		sort.Strings(folders)
		latestReleaseFolder := strings.TrimSuffix(folders[len(folders)-1], "/")

		// 2. Fetch SUMS from the specific immutable folder
		specificBaseURL := fmt.Sprintf("%s/%s/%s", strat.BaseURL, codename, latestReleaseFolder)
		sumsURL := specificBaseURL + "/SHA512SUMS"

		sumsContent, err := fetchString(sumsURL)
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to fetch SUMS for %s %s\n", src.Name, ver, latestReleaseFolder)
			continue
		}

		// debian-12-genericcloud-amd64-20260225-2399.qcow2
		lineRegex := regexp.MustCompile(`([a-f0-9]{128})\s+\*?((debian-\d+-genericcloud-amd64.*\.qcow2))`)
		sumsMatch := lineRegex.FindStringSubmatch(sumsContent)
		if sumsMatch == nil {
			fmt.Printf("⚠️ [%s] No matching image found in SUMS for %s %s\n", src.Name, ver, latestReleaseFolder)
			continue
		}

		hash := sumsMatch[1]
		filename := sumsMatch[2]

		if filename == "" {
			continue
		}

		imgURL := specificBaseURL + "/" + filename
		size, _ := getRemoteSize(imgURL)

		results = append(results, image.Version{
			Version:      ver,
			Aliases:      []string{codename},
			Arch:         "amd64",
			URL:          imgURL,
			ChecksumType: "sha512",
			Checksum:     hash,
			ChecksumURL:  fmt.Sprintf("%s/%s/current/SHA512SUMS", strat.BaseURL, codename),
			ChecksumRegex: regexp.QuoteMeta(filename),
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
			ChecksumURL:  checksumURL,
			ChecksumRegex: "^" + regexp.QuoteMeta(hash) + "$",
			SizeBytes:    size,
			Format:       "qcow2",
		})
	}
	return results, nil
}

func fetchOpenSUSE(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions { // ver can be just "Latest" mapped to the latest snapshot
		// 1. Fetch directory listing
		content, err := fetchString(strat.BaseURL + "/")
		if err != nil {
			fmt.Printf("⚠️ [%s] Failed to list dir for %s: %v\n", src.Name, ver, err)
			continue
		}

		// Find snapshot file: openSUSE-Tumbleweed-Minimal-VM.x86_64-1.0.0-Cloud-Snapshot20260224.qcow2
		// Or similar. Let's use a safe regex that captures the snapshot timestamp
		re := regexp.MustCompile(`(openSUSE-Tumbleweed-Minimal-VM\.x86_64(?:-\d+\.\d+\.\d+)?-Cloud-Snapshot(\d{8})\.qcow2)`)
		matches := re.FindAllStringSubmatch(content, -1)

		if len(matches) == 0 {
			fmt.Printf("⚠️ [%s] No snapshot file found for %s\n", src.Name, ver)
			continue
		}

		// Sort to find the latest snapshot
		sort.Slice(matches, func(i, j int) bool {
			return matches[i][2] < matches[j][2]
		})

		latestMatch := matches[len(matches)-1]
		filename := latestMatch[1]
		snapshotDate := latestMatch[2] // e.g. 20260224

		imgURL := strat.BaseURL + "/" + filename
		checksumURL := imgURL + ".sha256"

		// 2. Fetch Checksum
		hash, err := findChecksum(checksumURL, "", "sha256")
		if err != nil {
			fmt.Printf("⚠️ [%s] Checksum failed for %s: %v\n", src.Name, ver, err)
			continue
		}

		// 3. Get Size
		size, _ := getRemoteSize(imgURL)

		results = append(results, image.Version{
			Version:      "Snapshot" + snapshotDate,
			Aliases:      []string{ver}, // maps "Latest" to this version
			Arch:         "amd64",
			URL:          imgURL,
			ChecksumType: "sha256",
			Checksum:     hash,
			ChecksumURL:  checksumURL,
			ChecksumRegex: "^" + regexp.QuoteMeta(hash) + "$",
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

func fetchGithubRelease(src Source, strat Strategy) ([]image.Version, error) {
	if strat.Owner == "" || strat.Repo == "" {
		return nil, fmt.Errorf("github-release strategy requires owner and repo")
	}

	// 1. Fetch Releases
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", strat.Owner, strat.Repo)
	if strat.Tag != "" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", strat.Owner, strat.Repo, strat.Tag)
	}

	req, _ := http.NewRequest("GET", apiURL, nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var releases []githubRelease
	if strat.Tag != "" {
		var r githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode github release: %w", err)
		}
		releases = append(releases, r)
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to decode github releases list: %w", err)
		}
	}

	// 2. Process Releases
	var results []image.Version

	for _, rel := range releases {
		type group struct {
			parts       []string
			size        int64
			checksum    string
			chkType     string
			compression string // "zst" or "none"
		}
		// Key: image version string
		groups := make(map[string]*group)

		// Temporary map to track the "best" version name for each flavour in this release
		flavourToVersion := make(map[string]string)

		// Regex to parse asset name: flavour-<name>[-<version>][-amd64].qcow2[.<suffix>]
		assetRegex := regexp.MustCompile(`flavour-(.+?)(?:-(v\d+.*))?(?:-amd64)?\.qcow2(\..+)?`)

		// First pass: Find specific versions for each flavour
		for _, asset := range rel.Assets {
			matches := assetRegex.FindStringSubmatch(asset.Name)
			if matches == nil || matches[1] != src.Name {
				continue
			}
			if matches[2] != "" {
				flavourToVersion[matches[1]] = strings.TrimSuffix(matches[2], "-amd64")
			}
		}

		// Second pass: Categorize and group assets
		for _, asset := range rel.Assets {
			matches := assetRegex.FindStringSubmatch(asset.Name)
			if matches == nil || matches[1] != src.Name {
				continue
			}

			flavourName := matches[1]
			version := strings.TrimSuffix(matches[2], "-amd64")
			suffix := matches[3]

			// Fallback logic for version: Prefer specific version from filename, then release tag
			if version == "" {
				if best, ok := flavourToVersion[flavourName]; ok {
					version = best
				} else {
					version = rel.TagName
				}
			}

			if groups[version] == nil {
				groups[version] = &group{parts: []string{}, compression: "none"}
			}
			g := groups[version]

			if strings.HasSuffix(suffix, ".sha256") {
				g.chkType = "sha256"
				ch, _ := fetchString(asset.DownloadURL)
				g.checksum = strings.Fields(ch)[0]
			} else if strings.HasSuffix(suffix, ".sha512") {
				g.chkType = "sha512"
				ch, _ := fetchString(asset.DownloadURL)
				g.checksum = strings.Fields(ch)[0]
			} else {
				isZst := strings.Contains(suffix, ".zst")
				isPart := regexp.MustCompile(`\.(zst\.[a-z]{2}|\d{3})$`).MatchString(suffix) || suffix == ".zst"

				if isPart {
					if isZst && g.compression == "none" {
						g.parts = []string{}
						g.size = 0
						g.compression = "zst"
					}
					if isZst == (g.compression == "zst") {
						g.parts = append(g.parts, asset.DownloadURL)
						g.size += asset.Size
					}
				}
			}
		}

		for ver, g := range groups {
			if len(g.parts) == 0 {
				continue
			}

			sort.Strings(g.parts)
			results = append(results, image.Version{
				Version:      ver,
				Aliases:      []string{ver},
				Arch:         "amd64",
				URL:          g.parts[0],
				PartURLs:     g.parts,
				ChecksumType: g.chkType,
				Checksum:     g.checksum,
				SizeBytes:    g.size,
				Format:       "qcow2",
			})
		}
	}

	// Deduplicate: Keep only the latest version (first one found since we process releases in order)
	if len(results) > 0 {
		return []image.Version{results[0]}, nil
	}

	return nil, fmt.Errorf("no versions found for flavour %s", src.Name)
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}
