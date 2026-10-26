package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Josepavese/nido/v4/internal/build"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
)

type updateHTTPFixture struct {
	metadata *releaseFixture
	client   sysutil.HTTPClient
}

func (f updateHTTPFixture) Do(r *http.Request) (*http.Response, error) {
	if r.URL.String() == build.LatestReleaseURL || strings.HasSuffix(r.URL.Path, "/sums") {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 30*time.Second {
			return nil, fmt.Errorf("metadata request lacks bounded deadline")
		}
		return f.metadata.Do(r)
	}
	deadline, ok := r.Context().Deadline()
	if !ok || time.Until(deadline) <= 30*time.Second || time.Until(deadline) > 15*time.Minute {
		return nil, fmt.Errorf("archive request lacks slow-transfer deadline")
	}
	return f.client.Do(r)
}

func TestUpdaterSlowConnectionBeyondFormerTimeout(t *testing.T) {
	fixture, request := updaterFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(fixture.archive)))
		// Deliver 32 chunks at one-second intervals: real forward progress for
		// longer than the former client-wide timeout, with immediate cancellation.
		for chunk := 0; chunk < 32; chunk++ {
			if chunk > 0 {
				timer := time.NewTimer(time.Second)
				select {
				case <-r.Context().Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			_, _ = w.Write(fixture.archive[chunk*len(fixture.archive)/32 : (chunk+1)*len(fixture.archive)/32])
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()
	fixture.metadata.Assets[0].URL = server.URL + "/archive"
	client := NewUpdater(UpdaterOptions{}).(*nativeUpdater).client
	started := time.Now()
	result, err := fixtureUpdater(updateHTTPFixture{fixture, client}).Update(context.Background(), request)
	if err != nil || result.RegistryError != nil {
		t.Fatalf("slow update: %+v %v", result, err)
	}
	if time.Since(started) < 30*time.Second {
		t.Fatal("did not exercise former timeout")
	}
	installed, err := os.ReadFile(request.Executable)
	if err != nil || string(installed) != "NEW EXECUTABLE" {
		t.Fatalf("installation: %q %v", installed, err)
	}
	t.Logf("verified atomic update after %s streaming transfer", time.Since(started).Round(time.Millisecond))
}

func TestUpdaterInterruptedTransferResumesOrSafelyRestarts(t *testing.T) {
	for _, mode := range []string{"range", "ignored range", "invalid range", "repeated disconnect"} {
		t.Run(mode, func(t *testing.T) {
			fixture, request := updaterFixture(t)
			requests := 0
			split := len(fixture.archive) / 2
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests == 1 || mode == "repeated disconnect" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					defer conn.Close()
					_, _ = fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", len(fixture.archive))
					_, _ = conn.Write(fixture.archive[:split])
					return
				}
				if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", split) {
					t.Errorf("resume header: %q", r.Header.Get("Range"))
				}
				data := fixture.archive
				if mode != "ignored range" {
					first := split
					if mode == "invalid range" {
						first++
					}
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", first, len(data)-1, len(data)))
					w.WriteHeader(http.StatusPartialContent)
					data = data[split:]
				}
				_, _ = w.Write(data)
			}))
			defer server.Close()
			fixture.metadata.Assets[0].URL = server.URL + "/archive"
			result, err := fixtureUpdater(updateHTTPFixture{fixture, server.Client()}).Update(context.Background(), request)
			failure := mode == "invalid range" || mode == "repeated disconnect"
			if failure != (err != nil) {
				t.Fatalf("update: %+v %v", result, err)
			}
			want := "NEW EXECUTABLE"
			if failure {
				want = "OLD EXECUTABLE"
			}
			data, _ := os.ReadFile(request.Executable)
			if string(data) != want {
				t.Fatalf("installed %q", data)
			}
			expectedRequests := 2
			if mode == "repeated disconnect" {
				expectedRequests = 3
			}
			if requests != expectedRequests {
				t.Fatalf("requests: %d", requests)
			}
			scratch, _ := filepath.Glob(filepath.Join(filepath.Dir(request.Executable), ".nido-update-*"))
			if len(scratch) != 0 {
				t.Fatalf("scratch: %v", scratch)
			}
		})
	}
}

func TestUpdaterCancellationDuringStreamingPreservesInstallation(t *testing.T) {
	fixture, request := updaterFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(fixture.archive)))
		_, _ = w.Write(fixture.archive[:1])
		w.(http.Flusher).Flush()
		cancel()
		<-r.Context().Done()
	}))
	defer server.Close()
	fixture.metadata.Assets[0].URL = server.URL
	_, err := fixtureUpdater(updateHTTPFixture{fixture, server.Client()}).Update(ctx, request)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	data, _ := os.ReadFile(request.Executable)
	if string(data) != "OLD EXECUTABLE" {
		t.Fatalf("installed %q", data)
	}
	scratch, _ := filepath.Glob(filepath.Join(filepath.Dir(request.Executable), ".nido-update-*"))
	if len(scratch) != 0 {
		t.Fatalf("scratch: %v", scratch)
	}
}

func TestReleaseDownloadRejectsWrongSizeAndUnboundedBodies(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		t.Run(fmt.Sprint(chunked), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if chunked {
					w.(http.Flusher).Flush()
				}
				_, _ = io.WriteString(w, strings.Repeat("x", 100))
			}))
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "archive")
			if err := downloadFile(context.Background(), server.Client(), server.URL, dest, 5); err == nil {
				t.Fatal("accepted oversized archive")
			}
			for _, p := range []string{dest, dest + ".part"} {
				if info, err := os.Stat(p); err == nil && info.Size() > 6 {
					t.Fatalf("unbounded output: %d", info.Size())
				}
			}
		})
	}
}
