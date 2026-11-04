package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProgressFunc receives download progress. It may be called from the download
// goroutine and should return quickly.
type ProgressFunc func(downloaded, total int64)

// Download fetches an asset from the first working URL into dir and verifies its
// size and sha256. The returned path is only valid when the hash matched.
func Download(ctx context.Context, asset Asset, dir string, userAgent string, progress ProgressFunc) (string, error) {
	if asset.Name == "" {
		return "", fmt.Errorf("asset has no name")
	}
	if len(asset.URLs) == 0 {
		return "", fmt.Errorf("asset %s has no download URL", asset.Name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, asset.Name)
	var errs []string
	for _, url := range asset.URLs {
		path, err := downloadOne(ctx, url, dest, asset, userAgent, progress)
		if err == nil {
			return path, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", hostOf(url), err))
	}
	return "", fmt.Errorf("all download sources failed: %s", strings.Join(errs, "; "))
}

func downloadOne(ctx context.Context, url, dest string, asset Asset, userAgent string, progress ProgressFunc) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	// A hard cap protects against a stalled mirror; the caller's context still
	// cancels immediately when the app shuts down.
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	part := dest + ".part"
	file, err := os.Create(part)
	if err != nil {
		return "", err
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(part)
		}
	}()

	total := asset.Size
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	if progress != nil {
		writer = io.MultiWriter(file, hasher, &progressWriter{progress: progress, total: total})
		progress(0, total)
	}
	written, copyErr := io.Copy(writer, resp.Body)
	if progress != nil {
		// Always report the final byte count; the throttled writer may not have
		// fired for a small or very fast download.
		progress(written, total)
	}
	if copyErr != nil {
		err = copyErr
		return "", err
	}
	if asset.Size > 0 && written != asset.Size {
		err = fmt.Errorf("size mismatch: got %d want %d", written, asset.Size)
		return "", err
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if asset.SHA256 == "" {
		err = fmt.Errorf("asset has no sha256; refusing to install unverified download")
		return "", err
	}
	if !strings.EqualFold(got, asset.SHA256) {
		err = fmt.Errorf("sha256 mismatch: got %s want %s", got, asset.SHA256)
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	// Windows does not replace an existing destination with Rename.
	_ = os.Remove(dest)
	if err := os.Rename(part, dest); err != nil {
		return "", err
	}
	return dest, nil
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
	written  int64
	last     time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	now := time.Now()
	if now.Sub(p.last) >= 150*time.Millisecond {
		p.last = now
		p.progress(p.written, p.total)
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
