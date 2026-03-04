package downloader

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay706/gitf/internal/cache"
)

func newTestCache(dir string) (*cache.Cache, error) {
	return cache.New(dir)
}

// mockTransport implements http.RoundTripper for unit-level HTTP mocking.
type mockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func newMockClient(fn func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: &mockTransport{RoundTripFunc: fn}}
}

func TestDoWithRetry_Success(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(3))
	resp, err := d.doWithRetry(context.Background(), "https://example.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoWithRetry_RetriesOn500(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("server error")),
				Header:     http.Header{},
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(3))
	resp, err := d.doWithRetry(context.Background(), "https://example.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoWithRetry_NoRetryOn404(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(3))
	_, err := d.doWithRetry(context.Background(), "https://example.com/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for 404), got %d", calls)
	}
}

func TestDoWithRetry_RateLimit403(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			h := http.Header{}
			h.Set("X-RateLimit-Remaining", "0")
			h.Set("X-RateLimit-Reset", "1700000000")
			return &http.Response{
				StatusCode: 403,
				Body:       io.NopCloser(strings.NewReader("rate limit exceeded")),
				Header:     h,
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(1))
	resp, err := d.doWithRetry(context.Background(), "https://example.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return nil, ctx.Err()
	})

	d := New(WithHTTPClient(client), WithMaxRetries(3))
	_, err := d.doWithRetry(ctx, "https://example.com/test")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCollectFiles_EmptyDir(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("[]")),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithBaseURL("https://api.test"), WithMaxRetries(0))
	files, err := d.CollectFiles(context.Background(), &GitHubURLInfo{
		Owner: "o", Repo: "r", Branch: "main", Path: "empty",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestCollectFiles_RecursiveDirectories(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		var body string
		switch {
		case strings.HasSuffix(req.URL.Path, "/contents/root/sub"):
			body = `[
				{"name":"file2.txt","path":"root/sub/file2.txt","type":"file","size":20,"download_url":"http://dl/file2.txt","url":""}
			]`
		case strings.HasSuffix(req.URL.Path, "/contents/root"):
			body = `[
				{"name":"file1.txt","path":"root/file1.txt","type":"file","size":10,"download_url":"http://dl/file1.txt","url":"https://api.test/repos/o/r/contents/root/sub"},
				{"name":"sub","path":"root/sub","type":"dir","size":0,"download_url":"","url":"https://api.test/repos/o/r/contents/root/sub"}
			]`
		default:
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     http.Header{},
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithBaseURL("https://api.test"), WithMaxRetries(0))
	files, err := d.CollectFiles(context.Background(), &GitHubURLInfo{
		Owner: "o", Repo: "r", Branch: "main", Path: "root",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Name != "file1.txt" {
		t.Errorf("expected file1.txt, got %s", files[0].Name)
	}
	if files[1].Name != "file2.txt" {
		t.Errorf("expected file2.txt, got %s", files[1].Name)
	}
}

func TestStreamFile_SHAVerification_Pass(t *testing.T) {
	content := "hello, world"
	h := sha1.Sum([]byte(content))
	expectedSHA := hex.EncodeToString(h[:])

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(content)),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(0), WithVerifySHA(true))
	destPath := filepath.Join(t.TempDir(), "testfile.txt")

	if err := d.streamFile(context.Background(), "https://example.com/file", destPath, expectedSHA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch: got %q, want %q", string(data), content)
	}
}

func TestStreamFile_SHAVerification_Fail(t *testing.T) {
	content := "hello, world"

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(content)),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(0), WithVerifySHA(true))
	destPath := filepath.Join(t.TempDir(), "testfile.txt")

	err := d.streamFile(context.Background(), "https://example.com/file", destPath, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected integrity error, got nil")
	}

	var intErr *IntegrityError
	if !errors.As(err, &intErr) {
		t.Fatalf("expected *IntegrityError, got %T: %v", err, err)
	}

	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("expected file to be cleaned up after integrity failure")
	}
}

func TestStreamFile_SHASkippedWhenDisabled(t *testing.T) {
	content := "some data"

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(content)),
			Header:     http.Header{},
		}, nil
	})

	d := New(WithHTTPClient(client), WithMaxRetries(0), WithVerifySHA(false))
	destPath := filepath.Join(t.TempDir(), "testfile.txt")

	if err := d.streamFile(context.Background(), "https://example.com/file", destPath, "wrong-sha"); err != nil {
		t.Fatalf("expected no error when verify disabled: %v", err)
	}
}

func TestDoWithRetry_ETagCaching(t *testing.T) {
	calls := 0
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if etag := req.Header.Get("If-None-Match"); etag == `W/"test-etag"` {
			return &http.Response{
				StatusCode: 304,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		}
		h := http.Header{}
		h.Set("ETag", `W/"test-etag"`)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`[{"name":"a.txt"}]`)),
			Header:     h,
		}, nil
	})

	tmpDir := t.TempDir()
	cacheImport, err := newTestCache(tmpDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	d := New(WithHTTPClient(client), WithMaxRetries(0), WithCache(cacheImport))

	// First call: cache miss, should get 200 and store
	resp1, err := d.doWithRetry(context.Background(), "https://api.test/repos/o/r/contents/dir")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	if string(body1) != `[{"name":"a.txt"}]` {
		t.Errorf("first call body: got %q", string(body1))
	}

	// Second call: should send If-None-Match, get 304, serve from cache
	resp2, err := d.doWithRetry(context.Background(), "https://api.test/repos/o/r/contents/dir")
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != `[{"name":"a.txt"}]` {
		t.Errorf("second call body: got %q", string(body2))
	}

	if calls != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", calls)
	}
}
