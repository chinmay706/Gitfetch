package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chinmay706/gitf/internal/downloader"
)

func testServer(dl *downloader.Downloader) *Server {
	return New(dl, WithPort(0))
}

func TestHealthEndpoint(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

func TestPreviewEndpoint_MissingURL(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var body errorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error == "" {
		t.Error("expected error message in response")
	}
}

func TestPreviewEndpoint_InvalidURL(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview?url=https://gitlab.com/foo/bar", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPreviewEndpoint_Success(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []map[string]any{
			{"name": "a.txt", "path": "docs/a.txt", "type": "file", "size": 10,
				"sha": "abc123", "download_url": "http://dl/a.txt", "url": ""},
		}
		json.NewEncoder(w).Encode(items)
	}))
	defer ghServer.Close()

	dl := downloader.New(
		downloader.WithBaseURL(ghServer.URL),
		downloader.WithMaxRetries(0),
	)
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/preview?url=https://github.com/owner/repo/tree/main/docs", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp previewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Owner != "owner" || resp.Repo != "repo" || resp.Branch != "main" {
		t.Errorf("unexpected info: %+v", resp)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	if resp.Files[0].SHA != "abc123" {
		t.Errorf("expected sha=abc123, got %q", resp.Files[0].SHA)
	}
	if resp.TotalSize != 10 {
		t.Errorf("expected total_size=10, got %d", resp.TotalSize)
	}
}

func TestDownloadEndpoint_MissingURL(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/download",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadEndpoint_StreamsZIP(t *testing.T) {
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/contents/"):
			items := []map[string]any{
				{"name": "hello.txt", "path": "folder/hello.txt", "type": "file", "size": 5,
					"sha": "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
					"download_url": "http://" + r.Host + "/raw/hello.txt", "url": ""},
			}
			json.NewEncoder(w).Encode(items)
		case strings.Contains(r.URL.Path, "/raw/"):
			io.WriteString(w, "hello")
		default:
			http.NotFound(w, r)
		}
	}))
	defer ghServer.Close()

	dl := downloader.New(
		downloader.WithBaseURL(ghServer.URL),
		downloader.WithMaxRetries(0),
	)
	srv := testServer(dl)

	body := `{"url":"https://github.com/owner/repo/tree/main/folder"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/download",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected Content-Type=application/zip, got %q", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty ZIP body")
	}
}

func TestCORSHeaders(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := New(dl, WithCORSOrigin("http://localhost:8505"))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:8505" {
		t.Errorf("expected CORS origin=http://localhost:8505, got %q", origin)
	}
}

func TestRequestIDHeader(t *testing.T) {
	dl := downloader.New(downloader.WithMaxRetries(0))
	srv := testServer(dl)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if id := w.Header().Get("X-Request-ID"); id == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}
