package builder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestOpenSUSEMirrorRecovery(t *testing.T) {
	const filename = "openSUSE-Tumbleweed-Minimal-VM.x86_64-1.0.0-Cloud-Snapshot20261017.qcow2"
	checksum := strings.Repeat("a", 64)
	for _, tc := range []struct {
		name, metadata string
		primarySize    string
		redirect       bool
		want           string
		heads          []string
	}{
		{name: "healthy primary", primarySize: "12345", want: "/" + filename, heads: []string{"/" + filename}},
		{name: "redirected 404 recovers in priority order", redirect: true, metadata: `<url priority="2">{base}/good/{file}</url><url priority="1">{base}/dead/{file}</url>`, want: "/good/" + filename, heads: []string{"/" + filename, "/selected/" + filename, "/dead/" + filename, "/good/" + filename}},
		{name: "unknown primary size", primarySize: "0", metadata: `<url>{base}/good/{file}</url>`, want: "/good/" + filename, heads: []string{"/" + filename, "/good/" + filename}},
		{name: "wrong size skipped", metadata: `<url>{base}/wrong-size/{file}</url><url>{base}/good/{file}</url>`, want: "/good/" + filename, heads: []string{"/" + filename, "/wrong-size/" + filename, "/good/" + filename}},
		{name: "unsafe wrong filename and duplicate mirrors skipped", metadata: `<url>http://invalid.test/{file}</url><url>https://invalid.test/wrong.qcow2</url><url>{base}/{file}</url><url>{base}/dead/{file}</url><url>{base}/dead/{file}</url><url>{base}/good/{file}</url>`, want: "/good/" + filename, heads: []string{"/" + filename, "/dead/" + filename, "/good/" + filename}},
		{name: "bounded attempts", metadata: `<url>{base}/dead/1/{file}</url><url>{base}/dead/2/{file}</url><url>{base}/dead/3/{file}</url><url>{base}/good/{file}</url>`, heads: []string{"/" + filename, "/dead/1/" + filename, "/dead/2/" + filename, "/dead/3/" + filename}},
		{name: "missing mirrors", heads: []string{"/" + filename}},
		{name: "checksum mismatch", metadata: "bad-hash", heads: []string{"/" + filename}},
		{name: "missing checksum", metadata: "missing-hash", heads: []string{"/" + filename}},
		{name: "wrong identity", metadata: "wrong-file", heads: []string{"/" + filename}},
		{name: "zero size", metadata: "zero-size", heads: []string{"/" + filename}},
		{name: "malformed metadata", metadata: "malformed", heads: []string{"/" + filename}},
		{name: "upstream outage", metadata: "outage", heads: []string{"/" + filename}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var base string
			var heads []string
			metadataRequests := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					heads = append(heads, r.URL.Path)
					switch {
					case r.URL.Path == "/"+filename && tc.redirect:
						http.Redirect(w, r, "/selected/"+filename, http.StatusFound)
					case r.URL.Path == "/"+filename && tc.primarySize != "":
						w.Header().Set("Content-Length", tc.primarySize)
					case r.URL.Path == "/good/"+filename:
						w.Header().Set("Content-Length", "12345")
					case r.URL.Path == "/wrong-size/"+filename:
						w.Header().Set("Content-Length", "9")
					default:
						http.NotFound(w, r)
					}
					return
				}
				switch r.URL.Path {
				case "/":
					fmt.Fprintf(w, `<a href="./%s">snapshot</a>`, filename)
				case "/" + filename + ".sha256":
					fmt.Fprintf(w, "%s  %s", checksum, filename)
				case "/" + filename + ".meta4":
					metadataRequests++
					if tc.metadata == "outage" {
						http.Error(w, "outage", http.StatusServiceUnavailable)
						return
					}
					if tc.metadata == "malformed" {
						fmt.Fprint(w, "<broken>")
						return
					}
					hash, name, size := checksum, filename, "12345"
					if tc.metadata == "bad-hash" {
						hash = strings.Repeat("b", 64)
					}
					if tc.metadata == "wrong-file" {
						name = "different.qcow2"
					}
					if tc.metadata == "zero-size" {
						size = "0"
					}
					hashXML := fmt.Sprintf(`<hash type="sha-256">%s</hash>`, hash)
					if tc.metadata == "missing-hash" {
						hashXML = ""
					}
					mirrors := strings.NewReplacer("{base}", base, "{file}", filename).Replace(tc.metadata)
					fmt.Fprintf(w, `<metalink xmlns="urn:ietf:params:xml:ns:metalink"><file name="%s"><size>%s</size>%s%s</file></metalink>`, name, size, hashXML, mirrors)
				default:
					t.Errorf("unexpected payload GET: %s", r.URL)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			base = server.URL
			versions, err := NewFetcher(server.Client()).Fetch(Source{Name: "opensuse"}, Strategy{Type: "opensuse-tumbleweed", BaseURL: base, Versions: []string{"Latest"}})
			if tc.want == "" {
				if err == nil {
					t.Fatalf("accepted invalid/unavailable upstream: %+v", versions)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if len(versions) != 1 || versions[0].URL != base+tc.want || versions[0].Checksum != checksum || versions[0].SizeBytes != 12345 || versions[0].Version != "Snapshot20261017" || versions[0].ChecksumURL != base+"/"+filename+".sha256" {
					t.Fatalf("unexpected version: %+v", versions)
				}
			}
			if !reflect.DeepEqual(heads, tc.heads) {
				t.Fatalf("HEADs %v, want %v", heads, tc.heads)
			}
			if tc.primarySize == "12345" && metadataRequests != 0 {
				t.Fatal("healthy primary should not need mirror metadata")
			}
		})
	}
}
