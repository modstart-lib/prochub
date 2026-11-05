package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProgressFunc receives download progress. It may be called from the download
// goroutine and should return quickly.
type ProgressFunc func(downloaded, total int64)

const (
	// DefaultDownloadAttempts is the number of full passes over all mirrors
	// before a download is reported as failed.
	DefaultDownloadAttempts = 4
	defaultRetryDelay       = 500 * time.Millisecond
	maxRetryDelay           = 5 * time.Second
	downloadTimeout         = 30 * time.Minute
)

// errDownloadTerminal marks a completed transfer that failed verification.
// Other mirrors are still tried in the same pass, but repeating the same bytes
// after a backoff cannot turn a checksum mismatch into a good download.
var errDownloadTerminal = errors.New("download content rejected")

// DownloadOptions controls resumable download behaviour.
type DownloadOptions struct {
	// UserAgent is sent with every request.
	UserAgent string
	// Progress receives byte counts; it may be nil.
	Progress ProgressFunc
	// Attempts is the number of passes over asset.URLs. Zero uses
	// DefaultDownloadAttempts. A pass resumes from the leftover .part file.
	Attempts int
	// RetryDelay is the base backoff between passes. Zero uses 500ms; it
	// doubles after every failed pass up to 5s. Tests may shorten it.
	RetryDelay time.Duration
}

// Download fetches an asset from the first working URL into dir and verifies its
// size and sha256. The returned path is only valid when the hash matched.
func Download(ctx context.Context, asset Asset, dir string, userAgent string, progress ProgressFunc) (string, error) {
	return DownloadWithOptions(ctx, asset, dir, DownloadOptions{
		UserAgent: userAgent,
		Progress:  progress,
	})
}

// DownloadWithOptions downloads with configurable retries. Every attempt tries
// all URLs and resumes from the existing .part file, so an interrupted transfer
// (app quit, network drop, proxy timeout) continues instead of starting over.
func DownloadWithOptions(ctx context.Context, asset Asset, dir string, opts DownloadOptions) (string, error) {
	if asset.Name == "" {
		return "", fmt.Errorf("asset has no name")
	}
	if len(asset.URLs) == 0 {
		return "", fmt.Errorf("asset %s has no download URL", asset.Name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	attempts := opts.Attempts
	if attempts <= 0 {
		attempts = DefaultDownloadAttempts
	}
	delay := opts.RetryDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}

	dest := filepath.Join(dir, asset.Name)
	var failures []string
	usedAttempts := 0
	for attempt := 1; attempt <= attempts; attempt++ {
		usedAttempts = attempt
		terminal := 0
		for _, url := range asset.URLs {
			path, err := downloadOne(ctx, url, dest, asset, opts.UserAgent, opts.Progress)
			if err == nil {
				return path, nil
			}
			failures = append(failures, fmt.Sprintf("%s: %v", hostOf(url), err))
			if errors.Is(err, errDownloadTerminal) {
				terminal++
			}
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
		}
		// Every mirror rejected the content (bad hash / wrong size / missing
		// checksum): another pass would produce the same result.
		if terminal == len(asset.URLs) {
			break
		}
		if attempt < attempts {
			if err := sleepContext(ctx, delay); err != nil {
				return "", err
			}
			delay *= 2
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
		}
	}
	if len(failures) > 8 {
		failures = failures[len(failures)-8:]
	}
	return "", fmt.Errorf("all download sources failed after %d attempt(s): %s", usedAttempts, strings.Join(failures, "; "))
}

func downloadOne(ctx context.Context, url, dest string, asset Asset, userAgent string, progress ProgressFunc) (string, error) {
	part := dest + ".part"

	// Resume from a previously interrupted transfer when possible.
	offset := partSize(part)
	if asset.Size > 0 {
		if offset > asset.Size {
			// A stale or foreign partial file; never resume from it.
			_ = os.Remove(part)
			offset = 0
		} else if offset == asset.Size {
			// A previous attempt already fetched every byte. Verify in place
			// before touching the network again.
			if err := VerifySHA256(part, asset.SHA256); err == nil {
				return activatePart(part, dest)
			}
			_ = os.Remove(part)
			offset = 0
		}
	}

	// Reflect the already downloaded bytes immediately so the UI shows the
	// resume position even while retries are in flight.
	if progress != nil && offset > 0 && asset.Size > 0 {
		progress(offset, asset.Size)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	// A hard cap protects against a stalled mirror; the caller's context still
	// cancels immediately when the app shuts down.
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var file *os.File
	var total int64
	switch resp.StatusCode {
	case http.StatusOK:
		// The server ignored Range (or we requested the whole file): start over
		// instead of appending a second copy.
		offset = 0
		file, err = os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		total = resp.ContentLength
	case http.StatusPartialContent:
		start, rangeTotal, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			// The server answered with a different range than requested.
			// Discard the partial file so the next attempt starts cleanly.
			_ = os.Remove(part)
			return "", fmt.Errorf("unexpected Content-Range %q (requested from %d)", resp.Header.Get("Content-Range"), offset)
		}
		file, err = os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		total = rangeTotal
	case http.StatusRequestedRangeNotSatisfiable:
		// The partial file is not a prefix of the current resource (or it is
		// already complete but unverified). Drop it and let the next attempt
		// download from zero.
		_ = os.Remove(part)
		return "", fmt.Errorf("server rejected resume range; restarting from zero")
	default:
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", fmt.Errorf("cannot open %s", part)
	}

	if total <= 0 {
		total = asset.Size
	}
	// The release metadata is authoritative: a proxy that reports a different
	// length must not produce a file with the wrong size.
	if asset.Size > 0 {
		total = asset.Size
	}
	written := offset
	if progress != nil {
		progress(written, total)
	}
	writer := io.Writer(file)
	if progress != nil {
		writer = io.MultiWriter(file, &progressWriter{progress: progress, total: total, start: offset})
	}
	n, copyErr := io.Copy(writer, resp.Body)
	written += n
	closeErr := file.Close()
	if progress != nil {
		// Always report the final byte count; the throttled writer may not have
		// fired for a small or very fast download.
		progress(written, total)
	}
	if copyErr != nil {
		// Keep the .part file: the next attempt resumes from the last byte.
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}

	info, err := os.Stat(part)
	if err != nil {
		return "", err
	}
	if asset.Size > 0 && info.Size() != asset.Size {
		// A complete response with the wrong length means the remote file
		// changed; the partial bytes cannot be trusted for a resume.
		_ = os.Remove(part)
		return "", fmt.Errorf("%w: size mismatch: got %d want %d", errDownloadTerminal, info.Size(), asset.Size)
	}
	if asset.SHA256 == "" {
		_ = os.Remove(part)
		return "", fmt.Errorf("%w: asset has no sha256; refusing to install unverified download", errDownloadTerminal)
	}
	if err := VerifySHA256(part, asset.SHA256); err != nil {
		// A corrupt partial must never seed the next attempt.
		_ = os.Remove(part)
		return "", fmt.Errorf("%w: %v", errDownloadTerminal, err)
	}
	return activatePart(part, dest)
}

// activatePart atomically moves the verified partial file into place.
func activatePart(part, dest string) (string, error) {
	// Windows does not replace an existing destination with Rename.
	_ = os.Remove(dest)
	if err := os.Rename(part, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func partSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

// parseContentRange parses "bytes 100-199/2000" (the total may be "*").
func parseContentRange(value string) (start, total int64, ok bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "bytes ") {
		return 0, 0, false
	}
	value = strings.TrimPrefix(value, "bytes ")
	slash := strings.IndexByte(value, '/')
	if slash < 0 {
		return 0, 0, false
	}
	rangePart := value[:slash]
	totalPart := value[slash+1:]
	dash := strings.IndexByte(rangePart, '-')
	if dash <= 0 {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(rangePart[:dash], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if totalPart != "*" {
		total, err = strconv.ParseInt(totalPart, 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	return start, total, true
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func VerifySHA256(path, want string) error {
	if want == "" {
		return fmt.Errorf("no expected sha256")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("sha256 mismatch: got %s want %s", got, want)
	}
	return nil
}

type progressWriter struct {
	progress ProgressFunc
	total    int64
	start    int64
	written  int64
	last     time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	now := time.Now()
	if now.Sub(p.last) >= 150*time.Millisecond {
		p.last = now
		p.progress(p.start+p.written, p.total)
	}
	return n, nil
}

func hostOf(raw string) string {
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	return raw
}
