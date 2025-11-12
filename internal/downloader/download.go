package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DownloadFolder downloads a GitHub folder and writes files directly to the output directory
func DownloadFolder(info *GitHubURLInfo, outputDirName string) error {
	// If output path exists and is a file, fail fast
	if fi, err := os.Stat(outputDirName); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("output path '%s' exists and is a file; choose a different -o", outputDirName)
		}
	}

	// Create a temporary directory for atomic operations
	tempDir := outputDirName + ".tmp"
	if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("could not create temp directory: %w", err)
	}

	// Collect all files from GitHub API
	initialAPIURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		info.Owner, info.Repo, info.Path, info.Branch)

	var allFiles []GitHubContent
	err := collectAllFiles(initialAPIURL, &allFiles)
	if err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	// Prepare base prefix to strip (ensure trailing slash if non-empty)
	basePrefix := info.Path
	if basePrefix != "" && !strings.HasSuffix(basePrefix, "/") {
		basePrefix = basePrefix + "/"
	}

	// Download files concurrently and write directly to disk
	err = downloadAndWriteFiles(allFiles, tempDir, basePrefix)
	if err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	// Remove existing output directory if it exists
	if err := os.RemoveAll(outputDirName); err != nil && !os.IsNotExist(err) {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to remove existing output directory: %w", err)
	}

	// Atomically rename temp directory to final output directory
	if err := os.Rename(tempDir, outputDirName); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to rename temp directory: %w", err)
	}

	return nil
}

func collectAllFiles(apiURL string, allFiles *[]GitHubContent) error {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	setGitHubHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Enhance 403 errors with guidance and rate limit info
		if resp.StatusCode == http.StatusForbidden {
			bodyBytes, _ := io.ReadAll(resp.Body)
			remaining := resp.Header.Get("X-RateLimit-Remaining")
			reset := resp.Header.Get("X-RateLimit-Reset")
			msg := fmt.Sprintf("github API responded with status: %s", resp.Status)
			if len(bodyBytes) > 0 {
				msg = fmt.Sprintf("%s - %s", msg, strings.TrimSpace(string(bodyBytes)))
			}
			if remaining == "0" && reset != "" {
				if sec, err := strconv.ParseInt(reset, 10, 64); err == nil {
					t := time.Unix(sec, 0)
					msg = fmt.Sprintf("%s. Rate limit reset at %s. Set GITHUB_TOKEN env var to increase limits.", msg, t.Local().Format(time.RFC1123))
				}
			} else {
				msg = msg + ". If this is due to rate limiting, set GITHUB_TOKEN to increase limits."
			}
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("github API responded with status: %s", resp.Status)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return err
	}

	for _, item := range contents {
		if item.Type == "file" {
			*allFiles = append(*allFiles, item)
		} else if item.Type == "dir" {
			// Recursively collect files from subdirectories
			if err := collectAllFiles(item.URL, allFiles); err != nil {
				return err
			}
		}
	}
	return nil
}

func downloadAndWriteFiles(files []GitHubContent, outputDir string, basePrefix string) error {
	// Download files concurrently and write directly to disk
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	// Limit concurrent downloads
	semaphore := make(chan struct{}, 10)

	for _, file := range files {
		wg.Add(1)
		go func(f GitHubContent) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Download file content
			content, err := downloadFileContent(f)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("failed to download %s: %w", f.Path, err):
				default:
				}
				return
			}

			// Calculate relative path
			rel := f.Path
			if basePrefix != "" && strings.HasPrefix(rel, basePrefix) {
				rel = strings.TrimPrefix(rel, basePrefix)
			}
			rel = strings.TrimLeft(rel, "/")

			// Create destination path
			destPath := filepath.Join(outputDir, rel)

			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				select {
				case errCh <- fmt.Errorf("failed to create directory for %s: %w", rel, err):
				default:
				}
				return
			}

			// Write to temporary file first for atomic operation
			tempFile := destPath + ".tmp"
			if err := os.WriteFile(tempFile, content, 0644); err != nil {
				select {
				case errCh <- fmt.Errorf("failed to write %s: %w", rel, err):
				default:
				}
				return
			}

			// Atomically rename to final destination
			if err := os.Rename(tempFile, destPath); err != nil {
				os.Remove(tempFile) // cleanup on failure
				select {
				case errCh <- fmt.Errorf("failed to finalize %s: %w", rel, err):
				default:
				}
				return
			}
		}(file)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Return first error if any
	if err, ok := <-errCh; ok {
		return err
	}

	return nil
}

func downloadFileContent(file GitHubContent) ([]byte, error) {
	req, err := http.NewRequest("GET", file.DownloadUrl, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}


