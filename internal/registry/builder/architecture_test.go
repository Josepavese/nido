package builder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDualArchitectureFetchAndAliases(t *testing.T) {
	hashes := map[string]string{"amd64": strings.Repeat("a", 64), "arm64": strings.Repeat("b", 64)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "12345")
			return
		}
		for arch, hash := range hashes {
			fmt.Fprintf(w, "%s  image-%s.qcow2\n", hash, arch)
		}
	}))
	defer server.Close()
	strat := Strategy{Architectures: []string{"amd64", "arm64"}, Type: "generic", BaseURL: server.URL, Versions: []string{"1.0"}, TemplateURL: "{base_url}/image-{arch}.qcow2", ChecksumURL: "{base_url}/SHA256SUMS", ChecksumType: "sha256", Regex: `image-{arch}\.qcow2$`}
	catalog, _, _, err := Build(SourcesConfig{Sources: []Source{{Name: "fixture", Strategies: []Strategy{strat}}}}, nil, NewFetcher(server.Client()), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for arch, hash := range hashes {
		_, v, err := catalog.FindImage("fixture", "1.0", arch)
		if err != nil || v.Checksum != hash || !strings.Contains(v.URL, arch) {
			t.Fatalf("variant %s: %#v %v", arch, v, err)
		}
	}
	if len(catalog.Images[0].Versions) != 2 {
		t.Fatal("one variant replaced the other")
	}
}
