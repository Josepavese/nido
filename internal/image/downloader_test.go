package image

import (
	"errors"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransferResumeFailuresPreservePartialAndFinal(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		rangeHeader string
	}{{"range rejected", 416, ""}, {"wrong offset", 206, "bytes 1-3/6"}, {"malformed range", 206, "invalid"}} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests > 5 {
					http.Error(w, "stop old recursive implementation", 500)
					return
				}
				w.Header().Set("Content-Range", tc.rangeHeader)
				w.WriteHeader(tc.status)
				fmt.Fprint(w, "def")
			}))
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "image")
			os.WriteFile(dest, []byte("ORIGINAL"), 0600)
			os.WriteFile(dest+".part", []byte("abc"), 0600)
			if err := NewTransfer(TransferOptions{OnProgress: nil}).Download(server.URL, dest, 6); err == nil {
				t.Fatal("invalid resume accepted")
			}
			if requests != 1 {
				t.Fatalf("unbounded retry: %d requests", requests)
			}
			for path, want := range map[string]string{dest: "ORIGINAL", dest + ".part": "abc"} {
				b, err := os.ReadFile(path)
				if err != nil || string(b) != want {
					t.Fatalf("lost data: %s %s %v", path, b, err)
				}
			}
		})
	}
}
func TestTransferValidResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=3-" {
			t.Errorf("resume request: %s", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 3-5/6")
		w.WriteHeader(206)
		fmt.Fprint(w, "def")
	}))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "image")
	os.WriteFile(dest+".part", []byte("abc"), 0600)
	if err := NewTransfer(TransferOptions{OnProgress: nil}).Download(server.URL, dest, 6); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "abcdef" {
		t.Fatalf("corrupt resume: %s", b)
	}
}
func TestTransferUnknownLengthHasNonnegativeProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.(http.Flusher).Flush(); fmt.Fprint(w, "image") }))
	defer server.Close()
	if err := NewTransfer(TransferOptions{OnProgress: func(current, total int64) {
		if current < 0 || total < 0 {
			t.Errorf("invalid progress: %d/%d", current, total)
		}
	}}).Download(server.URL, filepath.Join(t.TempDir(), "image"), 0); err != nil {
		t.Fatal(err)
	}
}
func TestMultipartFailureIsAtomicAndRetiresOwnedScratch(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "image")
	os.WriteFile(dest, []byte("ORIGINAL"), 0600)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/2" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "part1")
	}))
	defer server.Close()
	if err := NewTransfer(TransferOptions{OnProgress: nil}).DownloadMultiPart([]string{server.URL + "/1", server.URL + "/2"}, dest, 10); err == nil {
		t.Fatal("failed multipart reported success")
	}
	entries, _ := os.ReadDir(root)
	b, _ := os.ReadFile(dest)
	if len(entries) != 1 || string(b) != "ORIGINAL" {
		t.Fatalf("lost final image or retained scratch: %v %s", entries, b)
	}
}
func TestTarDecompressionDoesNotOverwriteOtherImages(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("native tar backend unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	images := filepath.Join(root, "images")
	os.Mkdir(source, 0755)
	os.Mkdir(images, 0755)
	os.WriteFile(filepath.Join(source, "disk.raw"), []byte("NEW IMAGE"), 0600)
	os.WriteFile(filepath.Join(source, "existing.qcow2"), []byte("OVERWRITE ATTEMPT"), 0600)
	existing := filepath.Join(images, "existing.qcow2")
	os.WriteFile(existing, []byte("KEEP"), 0600)
	archive := filepath.Join(images, "download.tar.xz")
	if out, err := exec.Command("tar", "-cJf", archive, "-C", source, "disk.raw", "existing.qcow2").CombinedOutput(); err != nil {
		t.Fatalf("native tar fixture: %s %v", out, err)
	}
	dest := filepath.Join(images, "new.qcow2")
	if err := NewTransfer(TransferOptions{OnProgress: nil}).Decompress(archive, dest); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{existing: "KEEP", dest: "NEW IMAGE"} {
		b, err := os.ReadFile(path)
		if err != nil || string(b) != want {
			t.Fatalf("unexpected artifact: %s %s %v", path, b, err)
		}
	}
	entries, _ := os.ReadDir(images)
	if len(entries) != 3 {
		t.Fatalf("extracted unrelated archive members: %v", entries)
	}
}
func TestFailedDecompressionPreservesDestination(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "bad.tar.xz")
	dest := filepath.Join(root, "image")
	os.WriteFile(src, []byte("invalid archive"), 0600)
	os.WriteFile(dest, []byte("KEEP"), 0600)
	if err := NewTransfer(TransferOptions{OnProgress: nil}).Decompress(src, dest); err == nil {
		t.Fatal("invalid archive accepted")
	}
	b, _ := os.ReadFile(dest)
	entries, _ := os.ReadDir(root)
	if string(b) != "KEEP" || len(entries) != 2 {
		t.Fatalf("lost data or left scratch: %s %v", b, entries)
	}
}

// A consumer depends only on Transfer, rather than a concrete downloader type.
type fixtureTransfer struct{}

func (fixtureTransfer) Download(_ string, dest string, _ int64) error {
	return os.WriteFile(dest, []byte("image"), 0600)
}
func (fixtureTransfer) DownloadMultiPart(_ []string, dest string, _ int64) error {
	return os.WriteFile(dest, []byte("image"), 0600)
}
func (fixtureTransfer) Decompress(string, string) error {
	return fmt.Errorf("unexpected decompression")
}
func TestPrepareLocalImageAcceptsSubstitute(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "image")
	if err := (fixtureTransfer{}).Prepare(Version{Arch: "amd64", URL: "https://example.test/image", Format: "qcow2"}, dest); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dest)
	if strings.TrimSpace(string(b)) != "image" {
		t.Fatal("substitute was not used")
	}
}

func TestSuccessfulTransferCannotOverwriteExistingImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "new image") }))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "image")
	os.WriteFile(dest, []byte("KEEP"), 0600)
	if err := NewTransfer(TransferOptions{OnProgress: nil}).Download(server.URL, dest, 9); err == nil {
		t.Fatal("existing artifact overwritten")
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "KEEP" {
		t.Fatalf("lost existing bytes: %s", data)
	}
}

func TestDownloadCannotWriteThroughPartialAlias(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			vm := filepath.Join(root, "vm.qcow2")
			dest := filepath.Join(root, "download")
			if err := os.WriteFile(vm, []byte("VM DATA"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "symlink" {
				err = os.Symlink(vm, dest+".part")
			} else {
				err = os.Link(vm, dest+".part")
			}
			if err != nil {
				t.Skipf("native link capability unavailable: %v", err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "new image") }))
			defer server.Close()
			if err := NewTransfer(TransferOptions{}).Download(server.URL, dest, 9); err == nil {
				t.Fatal("partial alias accepted")
			}
			data, _ := os.ReadFile(vm)
			if string(data) != "VM DATA" {
				t.Fatalf("modified VM data through partial alias: %s", data)
			}
		})
	}
}

func (f fixtureTransfer) Prepare(ver Version, dest string) error {
	return f.Download(ver.URL, dest, ver.SizeBytes)
}

func TestDownloadReservationRejectsConcurrentWriter(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "image")
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; fmt.Fprint(w, "image") }))
	defer server.Close()
	done := make(chan error, 1)
	go func() { done <- NewTransfer(TransferOptions{}).Download(server.URL, dest, 5) }()
	<-entered
	err := NewTransfer(TransferOptions{}).Download(server.URL, dest, 5)
	close(release)
	firstErr := <-done
	if !errors.Is(err, sysutil.ErrResourceBusy) {
		t.Fatalf("concurrent writer admitted: %v", err)
	}
	if firstErr != nil {
		t.Fatal(firstErr)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 || entries[0].Name() != "image" {
		t.Fatalf("reservation residue: %v", entries)
	}
}
func TestPrepareVerifiesBeforePublication(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		t.Run(fmt.Sprint(multipart), func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "image")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "CORRUPT") }))
			defer server.Close()
			ver := Version{Arch: "amd64", URL: server.URL, Checksum: strings.Repeat("0", 64), ChecksumType: "sha256"}
			if multipart {
				ver.PartURLs = []string{server.URL}
			}
			if err := NewTransfer(TransferOptions{}).Prepare(ver, dest); err == nil {
				t.Fatal("bad checksum accepted")
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatalf("unverified bytes published: %v", err)
			}
			entries, _ := os.ReadDir(root)
			for _, entry := range entries {
				if entry.Name() != "image.part" {
					t.Fatalf("unexpected scratch: %s", entry.Name())
				}
			}
		})
	}
}
