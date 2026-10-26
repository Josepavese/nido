package lifecycle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Josepavese/nido/v4/internal/build"
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type releaseFixture struct {
	metadata    build.Release
	archive     []byte
	checksum    string
	requests    []string
	failArchive bool
}

func (f *releaseFixture) Do(r *http.Request) (*http.Response, error) {
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	f.requests = append(f.requests, r.URL.String())
	status := 200
	var data []byte
	switch r.URL.String() {
	case build.LatestReleaseURL:
		data, _ = json.Marshal(f.metadata)
	case "https://example.test/archive":
		data = f.archive
		if f.failArchive {
			status = 503
		}
	case "https://example.test/sums":
		data = []byte(f.checksum)
	default:
		return nil, fmt.Errorf("unexpected request: %s", r.URL)
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(data)), Request: r}, nil
}
func updaterFixture(t *testing.T) (*releaseFixture, UpdateRequest) {
	t.Helper()
	root := t.TempDir()
	asset, binary := build.NativeReleaseAsset()
	archive := filepath.Join(root, asset)
	files := map[string][]byte{"package/" + binary: []byte("NEW EXECUTABLE"), "package/registry/images.json": []byte(`{"schema_version":"1","images":[]}`)}
	var err error
	if strings.HasSuffix(asset, ".zip") {
		err = writeZipFiles(archive, files)
	} else {
		err = writeTarGzFiles(archive, files)
	}
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	client := &releaseFixture{archive: data, checksum: fmt.Sprintf("%x  %s\n", sum, asset), metadata: build.Release{TagName: "v99.0.0", Assets: []build.ReleaseAsset{{Name: asset, URL: "https://example.test/archive", Size: int64(len(data))}, {Name: "SHA256SUMS", URL: "https://example.test/sums"}}}}
	target := filepath.Join(root, binary)
	if err := os.WriteFile(target, []byte("OLD EXECUTABLE"), 0755); err != nil {
		t.Fatal(err)
	}
	return client, UpdateRequest{CurrentVersion: "v1.0.0", Executable: target, NidoDir: filepath.Join(root, "data")}
}
func TestUpdaterFailurePreservesInstalledBinaryAndRetiresScratch(t *testing.T) {
	for _, name := range []string{"download fails", "missing checksum", "bad checksum", "existing backup", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			client, request := updaterFixture(t)
			observed := []events.Event{}
			request.Events = func(e events.Event) { observed = append(observed, e) }
			ctx := context.Background()
			switch name {
			case "download fails":
				client.failArchive = true
			case "missing checksum":
				client.metadata.Assets = client.metadata.Assets[:1]
			case "bad checksum":
				client.checksum = "bad  " + client.metadata.Assets[0].Name
			case "existing backup":
				os.WriteFile(request.Executable+".bak", []byte("ACTIVE OLD SESSION"), 0755)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := fixtureUpdater(client).Update(ctx, request); err == nil {
				t.Fatal("failure reported success")
			}
			if len(observed) != 2 || observed[1].Kind != events.Failed {
				t.Fatalf("failed update events: %+v", observed)
			}
			data, err := os.ReadFile(request.Executable)
			if err != nil || string(data) != "OLD EXECUTABLE" {
				t.Fatalf("lost installed binary: %s %v", data, err)
			}
			if name == "existing backup" {
				data, _ := os.ReadFile(request.Executable + ".bak")
				if string(data) != "ACTIVE OLD SESSION" {
					t.Fatal("overwrote backup consumer")
				}
			}
			scratch, _ := filepath.Glob(filepath.Join(filepath.Dir(request.Executable), ".nido-update-*"))
			if len(scratch) != 0 {
				t.Fatalf("inactive scratch retained: %v", scratch)
			}
		})
	}
}
func TestUpdaterInstallsVerifiedArtifactAndPreservesData(t *testing.T) {
	client, request := updaterFixture(t)
	observed := []events.Event{}
	request.Events = func(e events.Event) { observed = append(observed, e) }
	sentinels := []string{filepath.Join(request.NidoDir, "vms", "fixture.qcow2"), filepath.Join(request.NidoDir, "config.env"), filepath.Join(request.NidoDir, "run", "fixture.json")}
	for _, p := range sentinels {
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte("KEEP"), 0600)
	}
	result, err := fixtureUpdater(client).Update(context.Background(), request)
	if err != nil || result.RegistryError != nil || result.Version != "v99.0.0" {
		t.Fatalf("update: %+v %v", result, err)
	}
	if len(observed) != 2 || observed[1].Kind != events.Completed {
		t.Fatalf("update events: %+v", observed)
	}
	for p, want := range map[string]string{request.Executable: "NEW EXECUTABLE", request.Executable + ".bak": "OLD EXECUTABLE", sentinels[0]: "KEEP", sentinels[1]: "KEEP", sentinels[2]: "KEEP"} {
		data, err := os.ReadFile(p)
		if err != nil || string(data) != want {
			t.Fatalf("%s: %s %v", p, data, err)
		}
	}
	count := 0
	for _, url := range client.requests {
		if url == build.LatestReleaseURL {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("release identity fetched %d times", count)
	}
	scratch, _ := filepath.Glob(filepath.Join(filepath.Dir(request.Executable), ".nido-update-*"))
	if len(scratch) != 0 {
		t.Fatalf("scratch left behind: %v", scratch)
	}
}

func TestUnsafeUninstallEmitsFailureWithoutTouchingData(t *testing.T) {
	got := []events.Event{}
	err := Uninstall("/", "", func(e events.Event) { got = append(got, e) })
	if err == nil || len(got) != 2 || got[0].Kind != events.Started || got[1].Kind != events.Failed {
		t.Fatalf("unsafe uninstall: %v %+v", err, got)
	}
}

func fixtureUpdater(client sysutil.HTTPClient) Updater {
	return NewUpdater(UpdaterOptions{Client: client, Prepare: func(context.Context, string, string) error { return nil }})
}

func TestUpdaterPreparesCandidateBeforeReplacingExecutable(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			client, request := updaterFixture(t)
			calls := 0
			updater := NewUpdater(UpdaterOptions{Client: client, Prepare: func(ctx context.Context, candidate, home string) error {
				calls++
				if lease, err := sysutil.AcquireResource(request.Executable + ".update.lock"); err == nil {
					lease.Close()
					t.Fatal("candidate preparation ran without exclusive update ownership")
				}
				old, _ := os.ReadFile(request.Executable)
				next, _ := os.ReadFile(candidate)
				if string(old) != "OLD EXECUTABLE" || string(next) != "NEW EXECUTABLE" || home != request.NidoDir {
					t.Fatal("preparation identity/order invalid")
				}
				if fail {
					return fmt.Errorf("fixture migration failed")
				}
				return nil
			}})
			_, err := updater.Update(context.Background(), request)
			if (err != nil) != fail || calls != 1 {
				t.Fatalf("prepare: %d %v", calls, err)
			}
			if fail {
				data, _ := os.ReadFile(request.Executable)
				if string(data) != "OLD EXECUTABLE" {
					t.Fatal("replaced binary after preparation failed")
				}
				if _, err := os.Stat(request.Executable + ".bak"); !os.IsNotExist(err) {
					t.Fatal("moved previous executable")
				}
			}
		})
	}
}
