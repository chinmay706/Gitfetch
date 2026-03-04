package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFolder_Integration(t *testing.T) {
	fileContents := map[string]string{
		"readme.md":       "# Hello World",
		"src/main.go":     "package main\n",
		"src/util/help.go": "package util\n",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/repo/contents/project":
			items := []GitHubContent{
				{Name: "readme.md", Path: "project/readme.md", Type: "file", Size: 13,
					DownloadURL: fmt.Sprintf("http://%s/raw/readme.md", r.Host), APIURL: ""},
				{Name: "src", Path: "project/src", Type: "dir", Size: 0,
					DownloadURL: "", APIURL: fmt.Sprintf("http://%s/repos/test/repo/contents/project/src", r.Host)},
			}
			json.NewEncoder(w).Encode(items)

		case "/repos/test/repo/contents/project/src":
			items := []GitHubContent{
				{Name: "main.go", Path: "project/src/main.go", Type: "file", Size: 14,
					DownloadURL: fmt.Sprintf("http://%s/raw/src/main.go", r.Host), APIURL: ""},
				{Name: "util", Path: "project/src/util", Type: "dir", Size: 0,
					DownloadURL: "", APIURL: fmt.Sprintf("http://%s/repos/test/repo/contents/project/src/util", r.Host)},
			}
			json.NewEncoder(w).Encode(items)

		case "/repos/test/repo/contents/project/src/util":
			items := []GitHubContent{
				{Name: "help.go", Path: "project/src/util/help.go", Type: "file", Size: 13,
					DownloadURL: fmt.Sprintf("http://%s/raw/src/util/help.go", r.Host), APIURL: ""},
			}
			json.NewEncoder(w).Encode(items)

		case "/raw/readme.md":
			fmt.Fprint(w, fileContents["readme.md"])
		case "/raw/src/main.go":
			fmt.Fprint(w, fileContents["src/main.go"])
		case "/raw/src/util/help.go":
			fmt.Fprint(w, fileContents["src/util/help.go"])

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	outputDir := filepath.Join(t.TempDir(), "output")

	dl := New(
		WithBaseURL(srv.URL),
		WithConcurrency(2),
		WithMaxRetries(0),
	)

	info := &GitHubURLInfo{Owner: "test", Repo: "repo", Branch: "main", Path: "project"}
	if err := dl.DownloadFolder(context.Background(), info, outputDir); err != nil {
		t.Fatalf("DownloadFolder failed: %v", err)
	}

	for relPath, expectedContent := range fileContents {
		fullPath := filepath.Join(outputDir, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", relPath, err)
			continue
		}
		if string(data) != expectedContent {
			t.Errorf("%s: got %q, want %q", relPath, string(data), expectedContent)
		}
	}
}

func TestDownloadFolder_404Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dl := New(WithBaseURL(srv.URL), WithMaxRetries(0))
	info := &GitHubURLInfo{Owner: "o", Repo: "r", Branch: "main", Path: "missing"}

	err := dl.DownloadFolder(context.Background(), info, filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestDownloadFolder_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block forever — context should cancel us
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	dl := New(WithBaseURL(srv.URL), WithMaxRetries(0))
	info := &GitHubURLInfo{Owner: "o", Repo: "r", Branch: "main", Path: "path"}

	err := dl.DownloadFolder(ctx, info, filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error on canceled context, got nil")
	}
}

func TestCollectFiles_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []GitHubContent{
			{Name: "a.txt", Path: "dir/a.txt", Type: "file", Size: 5, DownloadURL: "http://x/a", APIURL: ""},
			{Name: "b.txt", Path: "dir/b.txt", Type: "file", Size: 10, DownloadURL: "http://x/b", APIURL: ""},
		}
		json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	dl := New(WithBaseURL(srv.URL), WithMaxRetries(0))
	files, err := dl.CollectFiles(context.Background(), &GitHubURLInfo{
		Owner: "o", Repo: "r", Branch: "main", Path: "dir",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	if totalSize != 15 {
		t.Errorf("expected total size 15, got %d", totalSize)
	}
}
