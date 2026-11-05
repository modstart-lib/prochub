package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func resumePayload() []byte {
	return bytes.Repeat([]byte("0123456789abcdef"), 8192) // 128 KiB
}

func resumeSum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

type headerRecorder struct {
	mu    sync.Mutex
	items []string
}

func (r *headerRecorder) add(value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, value)
}

func (r *headerRecorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.items...)
}

func rangeStart(header string) int {
	// bytes=65536-
	value := strings.TrimPrefix(header, "bytes=")
	value = strings.TrimSuffix(value, "-")
	start, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return start
}

// TestDownloadResumesFromPartialFile pre-creates a .part file and expects the
// server to receive a Range request from that offset.
func TestDownloadResumesFromPartialFile(t *testing.T) {
	payload := resumePayload()
	half := len(payload) / 2
	var ranges headerRecorder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ranges.add(r.Header.Get("Range"))
		gotRange := r.Header.Get("Range")
		if gotRange != "" {
			start := rangeStart(gotRange)
			if start < 0 || start > len(payload) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(payload)-1, len(payload)))
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)-start))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(payload[start:])
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	asset := Asset{
		Name:   "QuotaPanel-resume.bin",
		Size:   int64(len(payload)),
		SHA256: resumeSum(payload),
		URLs:   []string{server.URL},
	}
	if err := os.WriteFile(filepath.Join(dir, asset.Name+".part"), payload[:half], 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := Download(context.Background(), asset, dir, "test", nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := ranges.all(); len(got) != 1 || got[0] != fmt.Sprintf("bytes=%d-", half) {
		t.Fatalf("ranges = %v, want [bytes=%d-]", got, half)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded %d bytes, payload differs", len(got))
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Fatalf(".part should be removed after success, err=%v", err)
	}
}

// TestDownloadRestartsWhenServerIgnoresRange covers proxies that return 200 for
// a Range request: the partial file must be truncated, not appended to.
func TestDownloadRestartsWhenServerIgnoresRange(t *testing.T) {
	payload := resumePayload()
	half := len(payload) / 2
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	asset := Asset{
		Name:   "QuotaPanel-restart.bin",
		Size:   int64(len(payload)),
		SHA256: resumeSum(payload),
		URLs:   []string{server.URL},
	}
	if err := os.WriteFile(filepath.Join(dir, asset.Name+".part"), payload[:half], 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := DownloadWithOptions(context.Background(), asset, dir, DownloadOptions{
		UserAgent:  "test",
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded %d bytes (duplicated prefix?), payload differs", len(got))
	}
}

// TestDownloadResumesAfterInterruptedTransfer simulates a dropped connection
// and verifies the retry sends a Range request from the received offset.
func TestDownloadResumesAfterInterruptedTransfer(t *testing.T) {
	payload := resumePayload()
	half := len(payload) / 2
	var attempts int32
	var ranges headerRecorder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		ranges.add(r.Header.Get("Range"))
		if n == 1 {
			// Declare the full length but send only the first half, then close.
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = w.Write(payload[:half])
			return
		}
		start := rangeStart(r.Header.Get("Range"))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(payload)-1, len(payload)))
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)-start))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start:])
	}))
	defer server.Close()

	dir := t.TempDir()
	asset := Asset{
		Name:   "QuotaPanel-interrupted.bin",
		Size:   int64(len(payload)),
		SHA256: resumeSum(payload),
		URLs:   []string{server.URL},
	}
	path, err := DownloadWithOptions(context.Background(), asset, dir, DownloadOptions{
		UserAgent:  "test",
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded %d bytes, payload differs", len(got))
	}
	if atomic.LoadInt32(&attempts) < 2 {
		t.Fatalf("attempts = %d, want at least 2", attempts)
	}
	gotRanges := ranges.all()
	if len(gotRanges) < 2 || gotRanges[1] != fmt.Sprintf("bytes=%d-", half) {
		t.Fatalf("ranges = %v, want second request from %d", gotRanges, half)
	}
}

// TestDownloadRetriesAllURLs verifies every pass retries the URLs after a
// transient failure instead of failing the whole update at once.
func TestDownloadRetriesAllURLs(t *testing.T) {
	payload := resumePayload()
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	asset := Asset{
		Name:   "QuotaPanel-retry.bin",
		Size:   int64(len(payload)),
		SHA256: resumeSum(payload),
		URLs:   []string{server.URL},
	}
	path, err := DownloadWithOptions(context.Background(), asset, dir, DownloadOptions{
		UserAgent:  "test",
		Attempts:   4,
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// TestDownloadDiscardsCorruptCompletePart makes sure a full-size but corrupt
// .part is deleted (not resumed) before the retry.
func TestDownloadDiscardsCorruptCompletePart(t *testing.T) {
	payload := resumePayload()
	var ranges headerRecorder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ranges.add(r.Header.Get("Range"))
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	asset := Asset{
		Name:   "QuotaPanel-corrupt.bin",
		Size:   int64(len(payload)),
		SHA256: resumeSum(payload),
		URLs:   []string{server.URL},
	}
	corrupt := bytes.Repeat([]byte{'x'}, len(payload))
	if err := os.WriteFile(filepath.Join(dir, asset.Name+".part"), corrupt, 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := DownloadWithOptions(context.Background(), asset, dir, DownloadOptions{
		UserAgent:  "test",
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("downloaded payload differs")
	}
	gotRanges := ranges.all()
	if len(gotRanges) != 1 || gotRanges[0] != "" {
		t.Fatalf("ranges = %v, want a single full download", gotRanges)
	}
}

func TestParseContentRange(t *testing.T) {
	start, total, ok := parseContentRange("bytes 100-199/2000")
	if !ok || start != 100 || total != 2000 {
		t.Fatalf("got start=%d total=%d ok=%v", start, total, ok)
	}
	start, total, ok = parseContentRange("bytes 10-19/*")
	if !ok || start != 10 || total != 0 {
		t.Fatalf("got start=%d total=%d ok=%v", start, total, ok)
	}
	if _, _, ok := parseContentRange("garbage"); ok {
		t.Fatal("garbage should not parse")
	}
}
