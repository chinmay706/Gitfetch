package downloader

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/chinmay706/gitf/internal/cache"
)

// HTTPClient abstracts *http.Client for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Progress reports download advancement to callers.
type Progress struct {
	TotalFiles     int
	CompletedFiles int
	CurrentFile    string
}

// Downloader fetches GitHub folder contents via the REST API.
type Downloader struct {
	client      HTTPClient
	baseURL     string
	token       string
	concurrency int
	maxRetries  int
	verifySHA   bool
	cache       *cache.Cache
	logger      *slog.Logger
	onProgress  func(Progress)
}

// Option configures a Downloader.
type Option func(*Downloader)

func WithHTTPClient(c HTTPClient) Option {
	return func(d *Downloader) { d.client = c }
}

func WithBaseURL(url string) Option {
	return func(d *Downloader) { d.baseURL = url }
}

func WithToken(token string) Option {
	return func(d *Downloader) { d.token = token }
}

func WithConcurrency(n int) Option {
	return func(d *Downloader) {
		if n > 0 {
			d.concurrency = n
		}
	}
}

func WithMaxRetries(n int) Option {
	return func(d *Downloader) {
		if n >= 0 {
			d.maxRetries = n
		}
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(d *Downloader) { d.logger = l }
}

func WithProgress(fn func(Progress)) Option {
	return func(d *Downloader) { d.onProgress = fn }
}

func WithVerifySHA(enable bool) Option {
	return func(d *Downloader) { d.verifySHA = enable }
}

func WithCache(c *cache.Cache) Option {
	return func(d *Downloader) { d.cache = c }
}

// New creates a Downloader with sensible defaults. Options override defaults.
func New(opts ...Option) *Downloader {
	d := &Downloader{
		baseURL:     "https://api.github.com",
		concurrency: 10,
		maxRetries:  3,
	}
	for _, opt := range opts {
		opt(d)
	}
	if d.client == nil {
		d.client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}
	if d.logger == nil {
		d.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if d.token == "" {
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			d.token = t
		} else if t := os.Getenv("GH_TOKEN"); t != "" {
			d.token = t
		}
	}
	return d
}

// HTTPTimeout returns the HTTP client timeout for external callers that
// need a standalone client matching the downloader's configuration.
func (d *Downloader) HTTPTimeout() time.Duration {
	if c, ok := d.client.(*http.Client); ok && c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}
