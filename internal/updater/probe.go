package updater

import (
	"context"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

const probeTimeout = 3 * time.Second

type probeResult struct {
	url  string
	dur  time.Duration
	ok   bool
	rank int
}

// OrderBySpeed probes every URL concurrently and returns the reachable ones
// fastest first, followed by the unreachable ones in their original order.
// Probe failures never remove a URL: the download still tries it as a fallback.
func OrderBySpeed(ctx context.Context, urls []string, userAgent string) []string {
	if len(urls) <= 1 {
		return append([]string(nil), urls...)
	}
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	results := make([]probeResult, len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			start := time.Now()
			ok := probeURL(probeCtx, url, userAgent)
			results[i] = probeResult{url: url, dur: time.Since(start), ok: ok, rank: i}
		}(i, url)
	}
	wg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].ok != results[j].ok {
			return results[i].ok
		}
		if results[i].ok && results[i].dur != results[j].dur {
			return results[i].dur < results[j].dur
		}
		return results[i].rank < results[j].rank
	})
	ordered := make([]string, 0, len(results))
	for _, result := range results {
		ordered = append(ordered, result.url)
	}
	return ordered
}

func probeURL(ctx context.Context, url, userAgent string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Range", "bytes=0-0")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	client := &http.Client{Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1)
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
