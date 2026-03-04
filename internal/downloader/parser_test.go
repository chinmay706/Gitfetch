package downloader

import (
	"testing"
)

func TestParseGithubURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    *GitHubURLInfo
		wantErr bool
	}{
		{
			name:   "valid URL with single-level path",
			rawURL: "https://github.com/spf13/cobra/tree/main/docs",
			want:   &GitHubURLInfo{Owner: "spf13", Repo: "cobra", Branch: "main", Path: "docs"},
		},
		{
			name:   "valid URL with deep path",
			rawURL: "https://github.com/facebook/react/tree/main/packages/react-dom/src",
			want:   &GitHubURLInfo{Owner: "facebook", Repo: "react", Branch: "main", Path: "packages/react-dom/src"},
		},
		{
			name:   "valid URL with non-main branch",
			rawURL: "https://github.com/owner/repo/tree/develop/some/path",
			want:   &GitHubURLInfo{Owner: "owner", Repo: "repo", Branch: "develop", Path: "some/path"},
		},
		{
			name:   "valid URL with trailing slash",
			rawURL: "https://github.com/owner/repo/tree/main/path/",
			want:   &GitHubURLInfo{Owner: "owner", Repo: "repo", Branch: "main", Path: "path"},
		},
		{
			name:   "valid URL branch only (no sub-path)",
			rawURL: "https://github.com/owner/repo/tree/main",
			want:   &GitHubURLInfo{Owner: "owner", Repo: "repo", Branch: "main", Path: ""},
		},
		{
			name:    "non-github host",
			rawURL:  "https://gitlab.com/owner/repo/tree/main/docs",
			wantErr: true,
		},
		{
			name:    "missing tree segment",
			rawURL:  "https://github.com/owner/repo/blob/main/file.go",
			wantErr: true,
		},
		{
			name:    "too few path segments",
			rawURL:  "https://github.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "empty URL",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "completely invalid URL",
			rawURL:  "://not-a-url",
			wantErr: true,
		},
		{
			name:   "URL with query parameters",
			rawURL: "https://github.com/owner/repo/tree/main/path?tab=readme",
			want:   &GitHubURLInfo{Owner: "owner", Repo: "repo", Branch: "main", Path: "path"},
		},
		{
			name:   "URL with fragment",
			rawURL: "https://github.com/owner/repo/tree/main/path#section",
			want:   &GitHubURLInfo{Owner: "owner", Repo: "repo", Branch: "main", Path: "path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGithubURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.want.Repo)
			}
			if got.Branch != tt.want.Branch {
				t.Errorf("Branch = %q, want %q", got.Branch, tt.want.Branch)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
		})
	}
}
