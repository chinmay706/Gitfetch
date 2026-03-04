package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GitHubContent represents a single item returned by the GitHub Contents API.
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
	APIURL      string `json:"url"`
}

func (d *Downloader) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "gitf-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if d.token != "" {
		req.Header.Set("Authorization", "token "+d.token)
	}
}

// fetchContents makes a single GitHub Contents API call and decodes the result.
func (d *Downloader) fetchContents(ctx context.Context, apiURL string) ([]GitHubContent, error) {
	resp, err := d.doWithRetry(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contents: %w", err)
	}
	defer resp.Body.Close()

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	return contents, nil
}

// CollectFiles recursively walks a GitHub directory tree and returns all files.
// This is the public entry point used by both DownloadFolder and dry-run mode.
func (d *Downloader) CollectFiles(ctx context.Context, info *GitHubURLInfo) ([]GitHubContent, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		d.baseURL, info.Owner, info.Repo, info.Path, info.Branch)
	return d.collectFilesRecursive(ctx, apiURL)
}

func (d *Downloader) collectFilesRecursive(ctx context.Context, apiURL string) ([]GitHubContent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	contents, err := d.fetchContents(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	var files []GitHubContent
	for _, item := range contents {
		switch item.Type {
		case "file":
			files = append(files, item)
		case "dir":
			sub, err := d.collectFilesRecursive(ctx, item.APIURL)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
		}
	}
	return files, nil
}
