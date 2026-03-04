package cache

import (
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	if err := c.Put("https://api.github.com/test", `W/"abc123"`, `[{"name":"file.txt"}]`); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	entry, ok := c.Get("https://api.github.com/test")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if entry.ETag != `W/"abc123"` {
		t.Errorf("ETag mismatch: got %q", entry.ETag)
	}
	if entry.Body != `[{"name":"file.txt"}]` {
		t.Errorf("Body mismatch: got %q", entry.Body)
	}
}

func TestGetMiss(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	_, ok := c.Get("https://nonexistent.url")
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestTTLExpiry(t *testing.T) {
	c, err := New(t.TempDir(), WithTTL(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	if err := c.Put("https://api.github.com/ttl", `"etag"`, "body"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("https://api.github.com/ttl")
	if ok {
		t.Error("expected cache miss after TTL expiry, got hit")
	}
}

func TestClear(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	_ = c.Put("https://a.com/1", `"e1"`, "body1")
	_ = c.Put("https://b.com/2", `"e2"`, "body2")

	if err := c.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if _, ok := c.Get("https://a.com/1"); ok {
		t.Error("expected miss after clear")
	}
	if _, ok := c.Get("https://b.com/2"); ok {
		t.Error("expected miss after clear")
	}
}

func TestDifferentKeysIndependent(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	_ = c.Put("https://a.com", `"e1"`, "body-a")
	_ = c.Put("https://b.com", `"e2"`, "body-b")

	a, ok := c.Get("https://a.com")
	if !ok || a.Body != "body-a" {
		t.Errorf("key a: got %v, ok=%v", a, ok)
	}

	b, ok := c.Get("https://b.com")
	if !ok || b.Body != "body-b" {
		t.Errorf("key b: got %v, ok=%v", b, ok)
	}
}
