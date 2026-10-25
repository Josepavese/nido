package builder

import (
	"encoding/json"
	"fmt"

	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type release struct {
	version  string
	codename string
	aliases  []string
	lts      bool
}

// numericLess compares upstream version/revision components numerically (r10 > r9).
func numericLess(a, b string) bool {
	numbers := regexp.MustCompile(`[0-9]+`)
	aa, bb := numbers.FindAllString(a, -1), numbers.FindAllString(b, -1)
	for i := 0; i < len(aa) && i < len(bb); i++ {
		x, _ := strconv.Atoi(aa[i])
		y, _ := strconv.Atoi(bb[i])
		if x != y {
			return x < y
		}
	}
	if len(aa) != len(bb) {
		return len(aa) < len(bb)
	}
	return a < b
}

func (f *UpstreamFetcher) discover(s Strategy) ([]release, error) {
	if s.Discovery == nil {
		if s.Type == "github-release" {
			return nil, nil
		}
		if len(s.Versions) == 0 {
			return nil, fmt.Errorf("%s requires versions or discovery", s.Type)
		}
		var out []release
		for _, v := range s.Versions {
			out = append(out, release{version: v})
		}
		return out, nil
	}
	if len(s.Versions) > 0 {
		return nil, fmt.Errorf("versions and discovery are mutually exclusive")
	}
	d := s.Discovery
	if d.URL == "" {
		return nil, fmt.Errorf("discovery URL is required")
	}
	var out []release
	if s.Type == "debian-cloud" {
		if len(d.Channels) == 0 {
			return nil, fmt.Errorf("discovery for Debian requires channels")
		}
		for _, channel := range d.Channels {
			// Only released channels are valid, never testing/unstable or arbitrary paths.
			if channel != "stable" && channel != "oldstable" {
				return nil, fmt.Errorf("unsupported Debian channel %q", channel)
			}
			text, err := f.fetchString(strings.TrimRight(d.URL, "/") + "/" + channel + "/Release")
			if err != nil {
				return nil, err
			}
			fields := paragraph(text)
			ver := strings.Split(fields["Version"], ".")[0]
			code := fields["Codename"]
			if !regexp.MustCompile(`^[0-9]+$`).MatchString(ver) || !regexp.MustCompile(`^[a-z]+$`).MatchString(code) || fields["Suite"] != channel {
				return nil, fmt.Errorf("invalid Debian %s Release metadata", channel)
			}
			out = append(out, release{version: ver, codename: code, aliases: []string{code, channel}})
		}
	} else {
		text, err := f.fetchString(d.URL)
		if err != nil {
			return nil, err
		}
		switch s.Type {
		case "ubuntu-cloud":
			out, err = parseUbuntuReleases(text, f.now(), s.architecture)
		case "alpine-cdn":
			out, err = parseAlpineReleases(text, f.now())
		case "generic":
			if d.Limit <= 0 {
				return nil, fmt.Errorf("directory discovery requires a positive limit")
			}
			var re *regexp.Regexp
			re, err = regexp.Compile(d.Pattern)
			if err == nil && re.NumSubexp() != 1 {
				err = fmt.Errorf("discovery pattern must capture exactly one version")
			}
			if err == nil {
				stable := regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+$`)
				for _, m := range re.FindAllStringSubmatch(text, -1) {
					if stable.MatchString(m[1]) {
						out = append(out, release{version: m[1]})
					}
				}
			}
		default:
			return nil, fmt.Errorf("discovery is not supported by %s", s.Type)
		}
		if err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	unique := []release{}
	for _, rel := range out {
		if !seen[rel.version] {
			seen[rel.version] = true
			unique = append(unique, rel)
		}
	}
	sort.Slice(unique, func(i, j int) bool { return numericLess(unique[j].version, unique[i].version) })
	if d.Limit > 0 && len(unique) > d.Limit {
		unique = unique[:d.Limit]
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("no supported releases discovered for %s", s.Type)
	}
	return unique, nil
}

func paragraph(s string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}

// The released SimpleStreams feed is authoritative for available, supported cloud images.
func parseUbuntuReleases(text string, now time.Time, arch string) ([]release, error) {
	var data struct {
		Products map[string]struct {
			Version   string `json:"version"`
			Series    string `json:"release"`
			Title     string `json:"release_title"`
			Arch      string `json:"arch"`
			OS        string `json:"os"`
			Supported bool   `json:"supported"`
			EOL       string `json:"support_eol"`
		} `json:"products"`
	}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return nil, err
	}
	var out []release
	for _, p := range data.Products {
		if p.Arch != arch || p.OS != "ubuntu" || !p.Supported {
			continue
		}
		end, err := time.Parse(time.DateOnly, p.EOL)
		if err != nil {
			return nil, err
		}
		if !end.After(now) {
			continue
		}
		if !regexp.MustCompile(`^[0-9]+\.[0-9]+$`).MatchString(p.Version) || !regexp.MustCompile(`^[a-z]+$`).MatchString(p.Series) {
			return nil, fmt.Errorf("invalid Ubuntu release metadata")
		}
		out = append(out, release{version: p.Version, codename: p.Series, aliases: []string{p.Series}, lts: strings.HasSuffix(p.Title, " LTS")})
	}
	return out, nil
}

func parseAlpineReleases(text string, now time.Time) ([]release, error) {
	var data struct {
		Branches []struct {
			Branch   string `json:"rel_branch"`
			Released string `json:"branch_date"`
			EOL      string `json:"eol_date"`
		} `json:"release_branches"`
	}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return nil, err
	}
	var out []release
	for _, b := range data.Branches {
		if b.Branch == "edge" {
			continue
		}
		if !regexp.MustCompile(`^v[0-9]+\.[0-9]+$`).MatchString(b.Branch) {
			return nil, fmt.Errorf("invalid Alpine branch %q", b.Branch)
		}
		start, err := time.Parse(time.DateOnly, b.Released)
		if err != nil {
			return nil, err
		}
		end, err := time.Parse(time.DateOnly, b.EOL)
		if err != nil {
			return nil, err
		}
		if !start.After(now) && end.After(now) {
			out = append(out, release{version: strings.TrimPrefix(b.Branch, "v")})
		}
	}
	return out, nil
}
