package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// doWithRetry executes a GET request with exponential backoff.
// It returns the response only for 2xx status codes; all other outcomes
// are either retried or surfaced as typed errors.
//
// When a cache is configured, it sends If-None-Match with the cached ETag.
// On 304 Not Modified (which is free and doesn't count against rate limits),
// it returns a synthetic response built from the cached body.
func (d *Downloader) doWithRetry(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	var cachedETag string

	if d.cache != nil {
		if entry, ok := d.cache.Get(url); ok {
			cachedETag = entry.ETag
		}
	}

	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(attempt - 1)
			if rlErr, ok := lastErr.(*RateLimitError); ok && !rlErr.ResetAt.IsZero() {
				if wait := time.Until(rlErr.ResetAt); wait > 0 && wait < 5*time.Minute {
					delay = wait
				}
			}
			d.logger.Debug("retrying request", "attempt", attempt, "delay", delay, "url", url)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		d.setHeaders(req)
		if cachedETag != "" {
			req.Header.Set("If-None-Match", cachedETag)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			d.logger.Debug("request failed", "error", err, "attempt", attempt)
			continue
		}

		// 304 Not Modified -- serve from cache (free, no rate limit cost)
		if resp.StatusCode == http.StatusNotModified && d.cache != nil {
			resp.Body.Close()
			if entry, ok := d.cache.Get(url); ok {
				d.logger.Debug("cache hit (304)", "url", url)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(entry.Body))),
					Header:     resp.Header,
				}, nil
			}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Store response in cache if an ETag is present.
			// Only cache API listing responses (JSON), not raw file downloads.
			if d.cache != nil {
				if etag := resp.Header.Get("ETag"); etag != "" {
					bodyBytes, readErr := io.ReadAll(resp.Body)
					resp.Body.Close()
					if readErr != nil {
						return nil, fmt.Errorf("failed to read response body: %w", readErr)
					}
					_ = d.cache.Put(url, etag, string(bodyBytes))
					d.logger.Debug("cached response", "url", url, "etag", etag)
					return &http.Response{
						StatusCode: resp.StatusCode,
						Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
						Header:     resp.Header,
					}, nil
				}
			}
			return resp, nil
		}

		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		body := strings.TrimSpace(string(bodyBytes))

		if isRateLimited(resp, body) {
			lastErr = &RateLimitError{
				StatusCode: resp.StatusCode,
				ResetAt:    parseRateLimitReset(resp),
				Message:    body,
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = &APIError{StatusCode: resp.StatusCode, Body: body}
			continue
		}

		return nil, &APIError{StatusCode: resp.StatusCode, Body: body}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", d.maxRetries+1, lastErr)
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func isRateLimited(resp *http.Response, body string) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if resp.StatusCode == http.StatusForbidden {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		if remaining == "0" {
			return true
		}
		lower := strings.ToLower(body)
		if strings.Contains(lower, "rate limit") || strings.Contains(lower, "abuse") {
			return true
		}
	}
	return false
}

func parseRateLimitReset(resp *http.Response) time.Time {
	raw := resp.Header.Get("X-RateLimit-Reset")
	if raw == "" {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}
