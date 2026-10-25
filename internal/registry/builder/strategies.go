package builder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/machine"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

type UpstreamFetcher struct {
	client sysutil.HTTPClient
	now    func() time.Time
}

const upstreamTimeout = 30 * time.Second
const maxMetadataBytes = 32 << 20

func NewFetcher(client sysutil.HTTPClient) *UpstreamFetcher {
	if client == nil {
		client = &http.Client{Timeout: upstreamTimeout}
	}
	return &UpstreamFetcher{client: client, now: time.Now}
}

// Fetch resolves current releases and dispatches through the provider boundary.
func (f *UpstreamFetcher) Fetch(src Source, strat Strategy) ([]image.Version, error) {
	if len(strat.Architectures) == 0 {
		return nil, fmt.Errorf("%s: strategy architectures must be declared", src.Name)
	}
	var result []image.Version
	for _, requested := range strat.Architectures {
		arch, err := machine.Normalize(requested)
		if err != nil {
			return nil, err
		}
		strat.architecture = arch
		variants, err := f.fetchArchitecture(src, strat)
		if err != nil {
			return nil, err
		}
		result = append(result, variants...)
	}
	return result, nil
}

func (f *UpstreamFetcher) fetchArchitecture(src Source, strat Strategy) ([]image.Version, error) {
	releases, err := f.discover(strat)
	if err != nil {
		return nil, err
	}
	strat.Versions = nil
	for _, rel := range releases {
		strat.Versions = append(strat.Versions, rel.version)
	}
	var result []image.Version
	switch strat.Type {
	case "generic":
		result, err = f.fetchGeneric(src, strat)
	case "ubuntu-cloud":
		result, err = f.fetchUbuntu(src, strat, releases)
	case "debian-cloud":
		result, err = f.fetchDebian(src, strat, releases)
	case "alpine-cdn":
		result, err = f.fetchAlpine(src, strat)
	case "opensuse-tumbleweed":
		result, err = f.fetchOpenSUSE(src, strat)
	case "github-release":
		result, err = f.fetchGithubRelease(src, strat)
	default:
		return nil, fmt.Errorf("unknown strategy type: %s", strat.Type)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(result, func(i, j int) bool { return numericLess(result[j].Version, result[i].Version) })
	if strat.Discovery != nil {
		ltsAssigned := false
		for i := range result {
			result[i].Aliases = nil
			for _, rel := range releases {
				if rel.version != result[i].Version {
					continue
				}
				result[i].Aliases = append(result[i].Aliases, rel.aliases...)
				if rel.lts && !ltsAssigned {
					result[i].Aliases = append(result[i].Aliases, "lts")
					ltsAssigned = true
				}
			}
			if i == 0 {
				result[i].Aliases = append(result[i].Aliases, "latest")
			}
		}
	}
	return result, nil
}

// --- Strategy Implementations ---

func (f *UpstreamFetcher) fetchGeneric(src Source, strat Strategy) ([]image.Version, error) {
	var results []image.Version

	for _, ver := range strat.Versions {
		// 1. Construct URLs
		replacer := strings.NewReplacer("{base_url}", strat.BaseURL, "{version}", ver, "{arch}", strat.architecture)
		imgURL := replacer.Replace(strat.TemplateURL)
		checksumURL := replacer.Replace(strat.ChecksumURL)
		regex := replacer.Replace(strat.Regex)

		format := strat.Format
		if format == "" {
			format = "qcow2"
		}

		// 2. Verify Image Existence (HEAD)
		size, err := f.getRemoteSize(imgURL)
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// 3. Fetch Checksum
		chk, err := f.findChecksum(checksumURL, regex, strat.ChecksumType)
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		results = append(results, image.Version{
			Version:       ver,
			Aliases:       []string{ver}, // TODO: Add aliases config if needed
			Arch:          strat.architecture,
			URL:           imgURL,
			ChecksumType:  strat.ChecksumType,
			Checksum:      chk,
			ChecksumURL:   checksumURL,
			ChecksumRegex: regex,
			SizeBytes:     size,
			Format:        format,
		})
	}
	return results, nil
}

func (f *UpstreamFetcher) fetchUbuntu(src Source, strat Strategy, releases []release) ([]image.Version, error) {
	var results []image.Version

	for _, rel := range releases {
		ver := rel.version
		// Ubuntu Cloud Images expose the same immutable builds under both
		// releases/<version>/ and releases/<codename>/. The numeric directory has
		// answered 403 while the codename directory kept serving them, so try both
		// instead of failing the whole catalog refresh.
		bases := []string{fmt.Sprintf("%s/%s", strat.BaseURL, ver)}
		if rel.codename != "" && rel.codename != ver {
			bases = append(bases, fmt.Sprintf("%s/%s", strat.BaseURL, rel.codename))
		}

		var content, base string
		var failures []string
		for _, candidate := range bases {
			body, err := f.fetchString(candidate + "/")
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			if !strings.Contains(body, "release-") {
				failures = append(failures, "no release directory")
				continue
			}
			content, base = body, candidate
			break
		}
		if base == "" {
			return nil, fmt.Errorf("%s:%s: upstream metadata unavailable or invalid (%s)", src.Name, ver, strings.Join(failures, "; "))
		}

		// Find latest release folder: href="release-YYYYMMDD/"
		re := regexp.MustCompile(`href="(release-\d{8}/)"`)
		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s:%s upstream metadata unavailable or invalid", src.Name, ver)
		}

		// Extract folder names and sort alphabetically (YYYYMMDD sorts correctly)
		var folders []string
		for _, m := range matches {
			folders = append(folders, m[1])
		}
		sort.Slice(folders, func(i, j int) bool { return numericLess(folders[i], folders[j]) })
		latestReleaseFolder := strings.TrimSuffix(folders[len(folders)-1], "/")

		// 2. Fetch SUMS from the specific immutable folder
		sumsURL := fmt.Sprintf("%s/%s/SHA256SUMS", base, latestReleaseFolder)
		sumsContent, err := f.fetchString(sumsURL)
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// Parse SUMS. Format: <hash> *<filename>
		lineRegex := regexp.MustCompile(`(?m)^([a-f0-9]{64})\s+\*?(ubuntu-` + regexp.QuoteMeta(ver) + `-server-cloudimg-` + regexp.QuoteMeta(strat.architecture) + `\.img)$`)
		sumsMatch := lineRegex.FindStringSubmatch(sumsContent)
		if sumsMatch == nil {
			return nil, fmt.Errorf("%s:%s upstream metadata unavailable or invalid", src.Name, ver)
		}

		hash := sumsMatch[1]
		filename := sumsMatch[2]
		imgURL := fmt.Sprintf("%s/%s/%s", base, latestReleaseFolder, filename)

		// 3. Get Size
		size, err := f.getRemoteSize(imgURL)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("%s: image size unavailable for %s: %v", src.Name, imgURL, err)
		}

		results = append(results, image.Version{
			Version:       ver,
			Aliases:       []string{},
			Arch:          strat.architecture,
			URL:           imgURL,
			ChecksumType:  "sha256",
			Checksum:      hash,
			ChecksumURL:   sumsURL,
			ChecksumRegex: regexp.QuoteMeta(filename),
			SizeBytes:     size,
			Format:        "qcow2",
		})
	}
	return results, nil
}

func (f *UpstreamFetcher) fetchDebian(src Source, strat Strategy, releases []release) ([]image.Version, error) {
	var results []image.Version

	for _, rel := range releases {
		ver, codename := rel.version, rel.codename
		if codename == "" {
			return nil, fmt.Errorf("discovery for Debian must supply a codename for %s", ver)
		}

		// 1. Fetch directory listing to find latest release folder
		baseURL := fmt.Sprintf("%s/%s", strat.BaseURL, codename)
		content, err := f.fetchString(baseURL + "/")
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// Find latest release folder: href="20260225-2399/"
		re := regexp.MustCompile(`href="(\d{8}-\d+/)"`)
		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s:%s upstream metadata unavailable or invalid", src.Name, ver)
		}

		// Extract folder names and sort
		var folders []string
		for _, m := range matches {
			folders = append(folders, m[1])
		}
		sort.Slice(folders, func(i, j int) bool { return numericLess(folders[i], folders[j]) })
		latestReleaseFolder := strings.TrimSuffix(folders[len(folders)-1], "/")

		// 2. Fetch SUMS from the specific immutable folder
		specificBaseURL := fmt.Sprintf("%s/%s/%s", strat.BaseURL, codename, latestReleaseFolder)
		sumsURL := specificBaseURL + "/SHA512SUMS"

		sumsContent, err := f.fetchString(sumsURL)
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// debian-12-genericcloud-amd64-20260225-2399.qcow2
		lineRegex := regexp.MustCompile(`([a-f0-9]{128})\s+\*?(debian-` + regexp.QuoteMeta(ver) + `-genericcloud-` + regexp.QuoteMeta(strat.architecture) + `[^\s]*\.qcow2)`)
		sumsMatch := lineRegex.FindStringSubmatch(sumsContent)
		if sumsMatch == nil {
			return nil, fmt.Errorf("%s:%s upstream metadata unavailable or invalid", src.Name, ver)
		}

		hash := sumsMatch[1]
		filename := sumsMatch[2]

		imgURL := specificBaseURL + "/" + filename
		size, err := f.getRemoteSize(imgURL)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("%s: image size unavailable for %s: %v", src.Name, imgURL, err)
		}

		results = append(results, image.Version{
			Version:       ver,
			Aliases:       []string{codename},
			Arch:          strat.architecture,
			URL:           imgURL,
			ChecksumType:  "sha512",
			Checksum:      hash,
			ChecksumURL:   sumsURL,
			ChecksumRegex: regexp.QuoteMeta(filename),
			SizeBytes:     size,
			Format:        "qcow2",
		})
	}
	return results, nil
}

func (f *UpstreamFetcher) fetchAlpine(src Source, strat Strategy) ([]image.Version, error) {
	if strat.architecture != machine.AMD64 {
		return nil, fmt.Errorf("this image source currently supports amd64 only")
	}
	var results []image.Version

	for _, ver := range strat.Versions {
		// URL: https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/cloud/
		baseURL := fmt.Sprintf("%s/v%s/releases/cloud", strat.BaseURL, ver)

		// 1. Fetch directory listing
		content, err := f.fetchString(baseURL + "/")
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// 2. Find latest image
		// Pattern: nocloud_alpine-3.21.0-x86_64-bios-cloudinit-r0.qcow2
		// Regex: href="(nocloud_alpine-<ver>(\.\d+)*-x86_64-bios-cloudinit-r\d+\.qcow2)"
		regexStr := fmt.Sprintf(`href="((?:nocloud|generic)_alpine-%s\.\d+-x86_64-bios-cloudinit-r\d+\.qcow2)"`, regexp.QuoteMeta(ver))
		re := regexp.MustCompile(regexStr)

		matches := re.FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s:%s no cloud-init BIOS image found", src.Name, ver)
		}
		sort.Slice(matches, func(i, j int) bool { return numericLess(matches[i][1], matches[j][1]) })

		// Take the last one (assuming higher version/revision is later in list)
		filename := matches[len(matches)-1][1]
		imgURL := baseURL + "/" + filename
		checksumURL := imgURL + ".sha512"

		// 3. Fetch Checksum (it's usually just the hash in the file)
		hash, err := f.findChecksum(checksumURL, "", "sha512")
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// 4. Get Size
		size, err := f.getRemoteSize(imgURL)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("%s: image size unavailable for %s: %v", src.Name, imgURL, err)
		}

		results = append(results, image.Version{
			Version:       ver,
			Aliases:       []string{ver},
			Arch:          strat.architecture,
			URL:           imgURL,
			ChecksumType:  "sha512",
			Checksum:      hash,
			ChecksumURL:   checksumURL,
			ChecksumRegex: "^" + regexp.QuoteMeta(hash) + "$",
			SizeBytes:     size,
			Format:        "qcow2",
		})
	}
	return results, nil
}

func (f *UpstreamFetcher) fetchOpenSUSE(src Source, strat Strategy) ([]image.Version, error) {
	if strat.architecture != machine.AMD64 {
		return nil, fmt.Errorf("this image source currently supports amd64 only")
	}
	var results []image.Version

	for _, ver := range strat.Versions { // ver can be just "Latest" mapped to the latest snapshot
		// 1. Fetch directory listing
		content, err := f.fetchString(strat.BaseURL + "/")
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// Find snapshot file: openSUSE-Tumbleweed-Minimal-VM.x86_64-1.0.0-Cloud-Snapshot20260224.qcow2
		// Or similar. Let's use a safe regex that captures the snapshot timestamp
		re := regexp.MustCompile(`(openSUSE-Tumbleweed-Minimal-VM\.x86_64(?:-\d+\.\d+\.\d+)?-Cloud-Snapshot(\d{8})\.qcow2)`)
		matches := re.FindAllStringSubmatch(content, -1)

		if len(matches) == 0 {
			return nil, fmt.Errorf("%s:%s upstream metadata unavailable or invalid", src.Name, ver)
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
		hash, err := f.findChecksum(checksumURL, "", "sha256")
		if err != nil {
			return nil, fmt.Errorf("%s:%s: %w", src.Name, ver, err)
		}

		// 3. Get Size
		resolvedURL, size, err := f.resolveOpenSUSEImage(imgURL, filename, hash)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src.Name, err)
		}

		results = append(results, image.Version{
			Version:       "Snapshot" + snapshotDate,
			Aliases:       []string{ver}, // maps "Latest" to this version
			Arch:          strat.architecture,
			URL:           resolvedURL,
			ChecksumType:  "sha256",
			Checksum:      hash,
			ChecksumURL:   checksumURL,
			ChecksumRegex: "^" + regexp.QuoteMeta(hash) + "$",
			SizeBytes:     size,
			Format:        "qcow2",
		})
	}
	return results, nil
}

// --- Helpers ---

func (f *UpstreamFetcher) getRemoteSize(url string) (int64, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		finalURL := req.URL
		if resp.Request != nil {
			finalURL = resp.Request.URL
		}
		return 0, fmt.Errorf("status %d at %s", resp.StatusCode, finalURL)
	}
	return resp.ContentLength, nil
}

func (f *UpstreamFetcher) fetchString(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMetadataBytes+1))
	if len(data) > maxMetadataBytes {
		return "", fmt.Errorf("metadata exceeds %d bytes", maxMetadataBytes)
	}
	return string(data), err
}

func (f *UpstreamFetcher) findChecksum(url, regexPattern, algo string) (string, error) {
	content, err := f.fetchString(url)
	if err != nil {
		return "", err
	}

	if regexPattern != "" {
		// Use regex to find line matching filename pattern, then extract hash
		// Standard format: HASH  FILENAME
		// We assume limits: SHA256 (64 chars), SHA512 (128 chars)

		scanner := bufio.NewScanner(strings.NewReader(content))
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return "", fmt.Errorf("invalid checksum pattern: %w", err)
		}

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

func (f *UpstreamFetcher) fetchGithubRelease(src Source, strat Strategy) ([]image.Version, error) {
	if strat.architecture != machine.AMD64 {
		return nil, fmt.Errorf("this image source currently supports amd64 only")
	}
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

	resp, err := f.client.Do(req)
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
				ch, _ := f.fetchString(asset.DownloadURL)
				if fields := strings.Fields(ch); len(fields) > 0 {
					g.checksum = fields[0]
				}
			} else if strings.HasSuffix(suffix, ".sha512") {
				g.chkType = "sha512"
				ch, _ := f.fetchString(asset.DownloadURL)
				if fields := strings.Fields(ch); len(fields) > 0 {
					g.checksum = fields[0]
				}
			} else {
				isZst := strings.Contains(suffix, ".zst")
				isPart := regexp.MustCompile(`\.(zst\.[a-z]{2}|\d{3})$`).MatchString(suffix) || suffix == ".zst"

				if isPart {
					if suffix == ".zst" && g.checksum == "" && strings.HasPrefix(asset.Digest, "sha256:") {
						g.chkType = "sha256"
						g.checksum = strings.TrimPrefix(asset.Digest, "sha256:")
					}
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
				Arch:         strat.architecture,
				URL:          g.parts[0],
				PartURLs:     g.parts,
				ChecksumType: g.chkType,
				Checksum:     g.checksum,
				SizeBytes:    g.size,
				Format:       "qcow2",
			})
		}
	}

	// Deduplicate deterministically: keep the highest version string for the flavour.
	if len(results) > 0 {
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Version > results[j].Version
		})
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
	Digest      string `json:"digest"`
}
