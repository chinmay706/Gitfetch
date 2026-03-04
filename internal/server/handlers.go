package server

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/chinmay706/gitf/internal/downloader"
)

// --- JSON response types ---

type previewResponse struct {
	Owner     string        `json:"owner"`
	Repo      string        `json:"repo"`
	Branch    string        `json:"branch"`
	Path      string        `json:"path"`
	Files     []previewFile `json:"files"`
	TotalSize int64         `json:"total_size"`
}

type previewFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	SHA  string `json:"sha"`
}

type downloadRequest struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing ?url= query parameter"})
		return
	}

	info, err := downloader.ParseGithubURL(rawURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	token := r.URL.Query().Get("token")
	dl := s.downloaderForRequest(token)

	files, err := dl.CollectFiles(r.Context(), info)
	if err != nil {
		s.writeDownloaderError(w, err)
		return
	}

	resp := previewResponse{
		Owner:  info.Owner,
		Repo:   info.Repo,
		Branch: info.Branch,
		Path:   info.Path,
	}
	for _, f := range files {
		resp.TotalSize += f.Size
		resp.Files = append(resp.Files, previewFile{
			Name: f.Name,
			Path: f.Path,
			Size: f.Size,
			SHA:  f.SHA,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing url field"})
		return
	}

	info, err := downloader.ParseGithubURL(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	dl := s.downloaderForRequest(req.Token)

	files, err := dl.CollectFiles(r.Context(), info)
	if err != nil {
		s.writeDownloaderError(w, err)
		return
	}
	if len(files) == 0 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "no files found at the specified path"})
		return
	}

	basePrefix := info.Path
	if basePrefix != "" && !strings.HasSuffix(basePrefix, "/") {
		basePrefix += "/"
	}

	folderName := info.Path
	if idx := strings.LastIndex(folderName, "/"); idx >= 0 {
		folderName = folderName[idx+1:]
	}
	if folderName == "" {
		folderName = info.Repo
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, folderName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	sess := &http.Client{Timeout: dl.HTTPTimeout()}
	for _, f := range files {
		rel := f.Path
		if basePrefix != "" {
			rel = strings.TrimPrefix(rel, basePrefix)
		}
		rel = strings.TrimLeft(rel, "/")

		reqFile, reqErr := http.NewRequestWithContext(r.Context(), http.MethodGet, f.DownloadURL, nil)
		if reqErr != nil {
			s.logger.Error("failed to create request", "path", rel, "error", reqErr)
			return
		}
		reqFile.Header.Set("User-Agent", "gitf-server")
		if req.Token != "" {
			reqFile.Header.Set("Authorization", "token "+req.Token)
		}

		resp, fetchErr := sess.Do(reqFile)
		if fetchErr != nil {
			s.logger.Error("failed to download file", "path", rel, "error", fetchErr)
			return
		}

		fw, zipErr := zw.Create(rel)
		if zipErr != nil {
			resp.Body.Close()
			s.logger.Error("zip create entry failed", "path", rel, "error", zipErr)
			return
		}
		_, _ = io.Copy(fw, resp.Body)
		resp.Body.Close()
	}
}

// --- Helpers ---

// downloaderForRequest creates a Downloader that may override the server's
// default token with a per-request token.
func (s *Server) downloaderForRequest(token string) *downloader.Downloader {
	if token != "" {
		return downloader.New(
			downloader.WithToken(token),
			downloader.WithLogger(s.logger),
		)
	}
	return s.dl
}

func (s *Server) writeDownloaderError(w http.ResponseWriter, err error) {
	switch err.(type) {
	case *downloader.RateLimitError:
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: err.Error()})
	case *downloader.APIError:
		apiErr := err.(*downloader.APIError)
		writeJSON(w, apiErr.StatusCode, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
