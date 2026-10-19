package builder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Josepavese/nido/internal/image"
)

func TestFutureReleasesAcrossProviders(t *testing.T) {
	hash := strings.Repeat("a", 64)
	longHash := strings.Repeat("b", 128)
	routes := map[string]string{
		"/ubuntu-metadata": `{"products":{
    "old":{"version":"24.04","release":"noble","release_title":"24.04 LTS","arch":"amd64","os":"ubuntu","supported":true,"support_eol":"2028-01-01"},
    "lts":{"version":"28.04","release":"futurelts","release_title":"28.04 LTS","arch":"amd64","os":"ubuntu","supported":true,"support_eol":"2033-01-01"},
    "latest":{"version":"28.10","release":"future","release_title":"28.10","arch":"amd64","os":"ubuntu","supported":true,"support_eol":"2029-07-01"},
    "preview":{"version":"30.04","release":"preview","arch":"amd64","os":"ubuntu","supported":false},
    "arm":{"version":"28.04","arch":"arm64","os":"ubuntu","supported":true}
  }}`,
		"/u/28.10/":                            `<a href="release-20281201/">build</a><a href="release-20281001/">older</a>`,
		"/u/28.04/":                            `<a href="release-20281201/">build</a>`,
		"/u/28.10/release-20281201/SHA256SUMS": hash + " *ubuntu-28.10-server-cloudimg-amd64.img\n",
		"/u/28.04/release-20281201/SHA256SUMS": hash + " *ubuntu-28.04-server-cloudimg-amd64.img\n",
		"/dists/stable/Release":                "Suite: stable\nVersion: 14.2\nCodename: forky\n",
		"/dists/oldstable/Release":             "Suite: oldstable\nVersion: 13.9\nCodename: trixie\n",
		"/d/forky/":                            `<a href="20281201-10000/">build</a>`,
		"/d/trixie/":                           `<a href="20281201-9999/">build</a>`,
		"/d/forky/20281201-10000/SHA512SUMS":   longHash + "  debian-14-genericcloud-amd64-20281201-10000.qcow2\n",
		"/d/trixie/20281201-9999/SHA512SUMS":   longHash + "  debian-13-genericcloud-amd64-20281201-9999.qcow2\n",
		"/alpine-metadata":                     `{"release_branches":[{"rel_branch":"edge"},{"rel_branch":"v3.9","branch_date":"2020-01-01","eol_date":"2030-01-01"},{"rel_branch":"v3.10","branch_date":"2028-01-01","eol_date":"2030-01-01"},{"rel_branch":"v3.8","branch_date":"2020-01-01","eol_date":"2028-01-01"},{"rel_branch":"v4.0","branch_date":"2030-01-01","eol_date":"2032-01-01"}]}`,
		"/a/v3.9/releases/cloud/":              `<a href="nocloud_alpine-3.9.1-x86_64-bios-cloudinit-r0.qcow2">old</a>`,
		"/a/v3.10/releases/cloud/":             `<a href="generic_alpine-3.10.10-x86_64-bios-cloudinit-r10.qcow2">new</a><a href="generic_alpine-3.10.9-x86_64-bios-cloudinit-r9.qcow2">old</a>`,
		"/a/v3.9/releases/cloud/nocloud_alpine-3.9.1-x86_64-bios-cloudinit-r0.qcow2.sha512":     longHash,
		"/a/v3.10/releases/cloud/generic_alpine-3.10.10-x86_64-bios-cloudinit-r10.qcow2.sha512": longHash,
		"/generic/":                   `<a href="v2028.9/">old</a><a href="v2028.10/">new</a><a href="v2028.10/">dup</a><a href="v2029.1-rc/">rc</a>`,
		"/generic/2028.10/SHA256SUMS": hash + "  image-amd64.img\n",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "12345")
			return
		}
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	fetcher := NewFetcher(server.Client())
	fetcher.now = func() time.Time { return time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC) }
	cases := []struct {
		name string
		s    Strategy
		want []string
	}{
		{"ubuntu", Strategy{Type: "ubuntu-cloud", BaseURL: server.URL + "/u", Discovery: &Discovery{URL: server.URL + "/ubuntu-metadata"}}, []string{"28.10", "28.04"}},
		{"debian", Strategy{Type: "debian-cloud", BaseURL: server.URL + "/d", Discovery: &Discovery{URL: server.URL + "/dists", Channels: []string{"stable", "oldstable"}}}, []string{"14", "13"}},
		{"alpine", Strategy{Type: "alpine-cdn", BaseURL: server.URL + "/a", Discovery: &Discovery{URL: server.URL + "/alpine-metadata"}}, []string{"3.10", "3.9"}},
		{"generic", Strategy{Type: "generic", BaseURL: server.URL + "/generic", TemplateURL: "{base_url}/{version}/image-amd64.img", ChecksumURL: "{base_url}/{version}/SHA256SUMS", ChecksumType: "sha256", Regex: `image-amd64\.img$`, Discovery: &Discovery{URL: server.URL + "/generic/", Pattern: `href="v([^"/]+)/"`, Limit: 1}}, []string{"2028.10"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			versions, err := fetcher.Fetch(Source{Name: tc.name}, tc.s)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			aliases := map[string]string{}
			for _, v := range versions {
				got = append(got, v.Version)
				if err := validateVersion(v); err != nil {
					t.Fatal(err)
				}
				for _, a := range v.Aliases {
					if _, ok := aliases[a]; ok {
						t.Fatalf("duplicate alias %s", a)
					}
					aliases[a] = v.Version
				}
				if (tc.name == "ubuntu" || tc.name == "debian") && strings.Contains(v.ChecksumURL, "/current/") {
					t.Fatal("mutable checksum URL")
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("versions %v, want %v", got, tc.want)
			}
			if aliases["latest"] != tc.want[0] {
				t.Fatal(aliases)
			}
			if tc.name == "ubuntu" && (aliases["lts"] != "28.04" || aliases["future"] != "28.10") {
				t.Fatal(aliases)
			}
			if tc.name == "debian" && (aliases["stable"] != "14" || !strings.Contains(versions[0].URL, "/forky/")) {
				t.Fatal(versions)
			}
			if tc.name == "alpine" && versions[1].Aliases != nil {
				t.Fatal("empty aliases must round-trip through JSON without timestamp drift")
			}
			if tc.name == "alpine" && !strings.Contains(versions[0].URL, "3.10.10") {
				t.Fatal(versions)
			}
		})
	}
}

func TestDiscoveryRejectsMissingMalformedAndAmbiguousInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			http.Error(w, "outage", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	f := NewFetcher(server.Client())
	cases := []Strategy{
		{Type: "ubuntu-cloud", Discovery: &Discovery{URL: server.URL + "/error"}},
		{Type: "ubuntu-cloud", Discovery: &Discovery{URL: server.URL}},
		{Type: "alpine-cdn", Discovery: &Discovery{URL: server.URL}},
		{Type: "generic", Discovery: &Discovery{URL: server.URL, Pattern: "[", Limit: 1}},
		{Type: "generic", Discovery: &Discovery{URL: server.URL, Pattern: "(a)(b)", Limit: 1}},
		{Type: "generic", Discovery: &Discovery{URL: server.URL, Pattern: "(a)", Limit: 0}},
		{Type: "ubuntu-cloud", Versions: []string{"24.04"}, Discovery: &Discovery{URL: server.URL}},
		{Type: "debian-cloud", Discovery: &Discovery{URL: server.URL, Channels: []string{"testing"}}},
	}
	for _, s := range cases {
		if _, err := f.Fetch(Source{Name: "bad"}, s); err == nil {
			t.Fatalf("accepted %#v", s)
		}
	}
}

type fixtureFetcher struct {
	versions []image.Version
	err      error
}

func (f fixtureFetcher) Fetch(Source, Strategy) ([]image.Version, error) { return f.versions, f.err }

func TestBuildIntegrityChangesAndStableTimestamp(t *testing.T) {
	v := image.Version{Version: "28.04", Aliases: []string{"latest", "lts"}, Arch: "amd64", URL: "https://example.test/build1/image", ChecksumType: "sha256", Checksum: strings.Repeat("a", 64), SizeBytes: 12345, Format: "qcow2"}
	config := SourcesConfig{Sources: []Source{{Name: "ubuntu", Strategies: []Strategy{{Type: "fixture"}}}}}
	now := time.Now().UTC()
	c, changed, _, err := Build(config, nil, fixtureFetcher{versions: []image.Version{v}}, now)
	if err != nil || len(changed) != 1 {
		t.Fatalf("%v %v", changed, err)
	}
	// Simulate persisted/reloaded JSON, including derived human sizes.
	data, _ := json.Marshal(c)
	var previous image.Catalog
	if err := json.Unmarshal(data, &previous); err != nil {
		t.Fatal(err)
	}
	next, changed, _, err := Build(config, &previous, fixtureFetcher{versions: []image.Version{v}}, now.Add(time.Hour))
	if err != nil || len(changed) != 0 || !next.UpdatedAt.Equal(now) {
		t.Fatalf("no-op drift: %v %v %v", next, changed, err)
	}
	v.URL = "https://example.test/build2/image"
	_, changed, _, err = Build(config, &previous, fixtureFetcher{versions: []image.Version{v}}, now.Add(time.Hour))
	if err != nil || len(changed) != 1 {
		t.Fatalf("changed bytes under same version not validated: %v %v", changed, err)
	}
	v.Checksum = "invalid"
	if _, _, _, err := Build(config, &previous, fixtureFetcher{versions: []image.Version{v}}, now); err == nil {
		t.Fatal("invalid checksum accepted")
	}
	failed := fixtureFetcher{err: fmt.Errorf("upstream unavailable")}
	if _, _, _, err := Build(config, &previous, failed, now); err == nil {
		t.Fatal("partial catalog accepted")
	}
	config.Sources[0].Optional = true
	next, _, warnings, err := Build(config, &previous, failed, now)
	if err != nil || len(warnings) != 1 || !reflect.DeepEqual(next.Images, previous.Images) {
		t.Fatalf("optional source data lost: %v %v", next, err)
	}
}
