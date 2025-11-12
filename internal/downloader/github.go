package downloader

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadUrl string `json:"download_url"`
	URL         string `json:"url"`
}

func getRepoContents(info *GitHubURLInfo) ([]GitHubContent, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		info.Owner, info.Repo, info.Path, info.Branch)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)

	}
	setGitHubHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API responded with status: %s", resp.Status)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	return contents, nil
}

// setGitHubHeaders sets common headers and optional auth for GitHub API
func setGitHubHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "gitf-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token != "" {
		// GitHub accepts either "token" or "Bearer" schemes
		req.Header.Set("Authorization", "token "+token)
	}
}
