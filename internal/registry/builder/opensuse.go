package builder

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
)

// Bound recovery when the geo-selected mirror has not received a new snapshot.
const maxOpenSUSEMirrorAttempts = 3

type openSUSEMetalink struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:metalink metalink"`
	Files   []struct {
		Name   string `xml:"name,attr"`
		Size   int64  `xml:"size"`
		Hashes []struct {
			Type  string `xml:"type,attr"`
			Value string `xml:",chardata"`
		} `xml:"hash"`
		Mirrors []struct {
			Priority int    `xml:"priority,attr"`
			URL      string `xml:",chardata"`
		} `xml:"url"`
	} `xml:"file"`
}

func (f *UpstreamFetcher) resolveOpenSUSEImage(imageURL, filename, checksum string) (string, int64, error) {
	size, headErr := f.getRemoteSize(imageURL)
	if headErr == nil && size > 0 {
		return imageURL, size, nil
	}
	// The official metalink supplies mirrors for this exact immutable artifact.
	// Never infer a different host or silently publish an older snapshot.
	metadata, err := f.fetchString(imageURL + ".meta4")
	if err != nil {
		return "", 0, fmt.Errorf("image unavailable at %s (HEAD: %v); metalink: %w", imageURL, headErr, err)
	}
	var document openSUSEMetalink
	if err := xml.Unmarshal([]byte(metadata), &document); err != nil {
		return "", 0, fmt.Errorf("invalid openSUSE metalink: %w", err)
	}
	if len(document.Files) != 1 || document.Files[0].Name != filename || document.Files[0].Size <= 0 {
		return "", 0, fmt.Errorf("openSUSE metalink does not identify %s with a positive size", filename)
	}
	file := document.Files[0]
	verified := false
	for _, hash := range file.Hashes {
		if hash.Type == "sha-256" {
			if strings.TrimSpace(hash.Value) != checksum {
				return "", 0, fmt.Errorf("openSUSE metalink checksum mismatch for %s", filename)
			}
			verified = true
		}
	}
	if !verified {
		return "", 0, fmt.Errorf("openSUSE metalink missing sha-256 for %s", filename)
	}
	sort.SliceStable(file.Mirrors, func(i, j int) bool { return file.Mirrors[i].Priority < file.Mirrors[j].Priority })
	seen := map[string]bool{imageURL: true}
	attempts := 0
	for _, mirror := range file.Mirrors {
		candidate := strings.TrimSpace(mirror.URL)
		u, err := url.Parse(candidate)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || path.Base(u.Path) != filename || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if attempts == maxOpenSUSEMirrorAttempts {
			break
		}
		attempts++
		size, err := f.getRemoteSize(candidate)
		if err == nil && size == file.Size {
			return candidate, size, nil
		}
	}
	return "", 0, fmt.Errorf("image unavailable at %s (HEAD: %v); no verified openSUSE mirror after %d attempts", imageURL, headErr, attempts)
}
