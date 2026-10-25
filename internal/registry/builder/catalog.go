package builder

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/Josepavese/nido/v4/internal/image"
)

// Build constructs a complete catalog before the command may replace the published file.
// Required upstream errors are fatal. Optional flavour sources retain prior published
// assets on an upstream failure; their absence is reported to the command boundary.
func Build(config SourcesConfig, previous *image.Catalog, fetcher Fetcher, now time.Time) (*image.Catalog, []string, []string, error) {
	result := &image.Catalog{SchemaVersion: "1", UpdatedAt: now.UTC(), Images: []image.Image{}}
	changed := []string{}
	warnings := []string{}
	names := map[string]bool{}
	if len(config.Sources) == 0 {
		return nil, nil, nil, fmt.Errorf("no sources configured")
	}
	for _, src := range config.Sources {
		if src.Name == "" || names[src.Name] || len(src.Strategies) == 0 {
			return nil, nil, nil, fmt.Errorf("invalid or duplicate source %q", src.Name)
		}
		names[src.Name] = true
		registry := src.Registry
		if registry == "" {
			registry = "official"
		}
		entry := image.Image{Name: src.Name, Registry: registry, Description: src.Description, Homepage: src.Homepage, SSHUser: src.SSHUser, Versions: []image.Version{}}
		var sourceErr error
		for _, strategy := range src.Strategies {
			versions, err := fetcher.Fetch(src, strategy)
			if err != nil {
				sourceErr = err
				break
			}
			if len(versions) == 0 {
				sourceErr = fmt.Errorf("no images found")
				break
			}
			entry.Versions = append(entry.Versions, versions...)
		}
		if sourceErr != nil {
			if !src.Optional {
				return nil, nil, nil, fmt.Errorf("%s: %w", src.Name, sourceErr)
			}
			warnings = append(warnings, fmt.Sprintf("optional source %s: %v", src.Name, sourceErr))
			if previous != nil {
				for _, old := range previous.Images {
					if old.Name == src.Name {
						result.Images = append(result.Images, old)
					}
				}
			}
			continue
		}
		versions := map[string]bool{}
		aliases := map[string]bool{}
		for i := range entry.Versions {
			v := &entry.Versions[i]
			v.SizeHuman = image.FormatBytes(v.SizeBytes)
			if versions[v.Version+":"+v.Arch] {
				return nil, nil, nil, fmt.Errorf("%s: duplicate version %s", src.Name, v.Version)
			}
			versions[v.Version+":"+v.Arch] = true
			if err := validateVersion(*v); err != nil {
				return nil, nil, nil, fmt.Errorf("%s:%s: %w", src.Name, v.Version, err)
			}
			for _, alias := range v.Aliases {
				if aliases[alias+":"+v.Arch] {
					return nil, nil, nil, fmt.Errorf("%s: ambiguous alias %q", src.Name, alias)
				}
				aliases[alias+":"+v.Arch] = true
			}
			var oldVersion *image.Version
			if previous != nil {
				_, oldVersion, _ = previous.FindImage(src.Name, v.Version, v.Arch)
			}
			// Image identity changes need validation even if the OS version is unchanged.
			if oldVersion == nil || oldVersion.URL != v.URL || oldVersion.Checksum != v.Checksum || oldVersion.ChecksumType != v.ChecksumType || oldVersion.Format != v.Format || oldVersion.Arch != v.Arch || oldVersion.SizeBytes != v.SizeBytes || !reflect.DeepEqual(oldVersion.PartURLs, v.PartURLs) {
				changed = append(changed, src.Name+":"+v.Version)
			}
		}
		result.Images = append(result.Images, entry)
	}
	if previous != nil && result.SchemaVersion == previous.SchemaVersion && reflect.DeepEqual(result.Images, previous.Images) {
		result.UpdatedAt = previous.UpdatedAt
	}
	return result, changed, warnings, nil
}

func validateVersion(v image.Version) error {
	u, err := url.Parse(v.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || v.Version == "" || v.Arch == "" || v.Format == "" || v.SizeBytes <= 0 {
		return fmt.Errorf("invalid image metadata")
	}
	expected := 0
	switch v.ChecksumType {
	case "sha256":
		expected = 32
	case "sha512":
		expected = 64
	default:
		return fmt.Errorf("unsupported checksum type %q", v.ChecksumType)
	}
	b, err := hex.DecodeString(v.Checksum)
	if err != nil || len(b) != expected {
		return fmt.Errorf("invalid %s checksum", v.ChecksumType)
	}
	return nil
}
