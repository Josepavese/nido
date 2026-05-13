package seediso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdomanski/iso9660"
)

func TestCreateWithGoWritesCIDATAImage(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "user-data"), []byte("#cloud-config\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "meta-data"), []byte("instance-id: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(tmp, "seed.iso")
	if err := createWithGo(out, source, "cidata"); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := iso9660.OpenImage(f)
	if err != nil {
		t.Fatal(err)
	}
	label, err := img.Label()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(label, "cidata") {
		t.Fatalf("unexpected ISO label %q", label)
	}

	root, err := img.RootDir()
	if err != nil {
		t.Fatal(err)
	}
	children, err := root.GetChildren()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, child := range children {
		seen[strings.ToLower(child.Name())] = true
	}
	if !seen["user-data"] || !seen["meta-data"] {
		t.Fatalf("expected user-data and meta-data in ISO, saw %v", seen)
	}
}
