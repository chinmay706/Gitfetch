package downloader

import (
	"fmt"
	"time"
)

// RateLimitError is returned when the GitHub API rate-limits a request.
type RateLimitError struct {
	StatusCode int
	ResetAt    time.Time
	Message    string
}

func (e *RateLimitError) Error() string {
	msg := fmt.Sprintf("rate limited (HTTP %d)", e.StatusCode)
	if !e.ResetAt.IsZero() {
		msg += fmt.Sprintf(", resets at %s", e.ResetAt.Local().Format(time.RFC1123))
	}
	if e.Message != "" {
		msg += ": " + e.Message
	}
	msg += ". Set GITHUB_TOKEN to increase limits"
	return msg
}

// APIError is returned for non-success, non-rate-limit GitHub API responses.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("GitHub API error (HTTP %d)", e.StatusCode)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}
