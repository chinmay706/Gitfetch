package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry holds a cached API response alongside its ETag and creation time.
type Entry struct {
	ETag      string    `json:"etag"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Cache is a file-based HTTP response cache keyed by URL.
// Each entry is stored as a JSON file named after the SHA-256 of the key.
type Cache struct {
	dir string
	ttl time.Duration
}

// Option configures a Cache.
type Option func(*Cache)

func WithTTL(ttl time.Duration) Option {
	return func(c *Cache) {
		if ttl > 0 {
			c.ttl = ttl
		}
	}
}

// New creates a file-based cache in dir. The directory is created if needed.
func New(dir string, opts ...Option) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	c := &Cache{dir: dir, ttl: 1 * time.Hour}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Get retrieves a cached entry. Returns nil, false on miss or expiry.
func (c *Cache) Get(key string) (*Entry, bool) {
	path := c.path(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}
	if time.Since(e.CreatedAt) > c.ttl {
		os.Remove(path)
		return nil, false
	}
	return &e, true
}

// Put stores an entry in the cache.
func (c *Cache) Put(key, etag, body string) error {
	e := Entry{
		ETag:      etag,
		Body:      body,
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(key), data, 0644)
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		os.Remove(filepath.Join(c.dir, entry.Name()))
	}
	return nil
}

func (c *Cache) path(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(h[:])+".json")
}
