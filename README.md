---
title: Gitfetch API
emoji: 📂
colorFrom: blue
colorTo: green
sdk: docker
pinned: false
---

<div align="center">

# Gitfetch

**Download any folder from GitHub — without cloning the entire repo.**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.11-3776AB?logo=python&logoColor=white)](https://python.org)
[![Streamlit](https://img.shields.io/badge/Streamlit-Frontend-FF4B4B?logo=streamlit&logoColor=white)](https://gitfetch.streamlit.app)
[![Docker](https://img.shields.io/badge/Docker-Backend-2496ED?logo=docker&logoColor=white)](https://huggingface.co/spaces/cheesechat/gitfetch-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

[**Try the Web App**](https://gitfetch.streamlit.app) · [**API Docs**](#api-endpoints) · [**CLI Usage**](#cli-usage) · [**Contributing**](#contributing)

</div>

---

## What is Gitfetch?

Gitfetch is a tool that lets you download a specific folder from any public GitHub repository. No `git clone`, no downloading the whole repo — just the folder you need, delivered as a direct download or a ZIP file.

It comes in three flavors:

| Interface | For whom | Link |
|-----------|----------|------|
| **Web App** | Anyone with a browser | [gitfetch.streamlit.app](https://gitfetch.streamlit.app) |
| **CLI Tool** | Developers on the terminal | `go install github.com/chinmay706/gitf@latest` |
| **REST API** | Services and scripts | [cheesechat-gitfetch-api.hf.space](https://cheesechat-gitfetch-api.hf.space/api/v1/health) |

---

## Tech Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **CLI & Backend** | ![Go](https://img.shields.io/badge/-Go-00ADD8?logo=go&logoColor=white&style=flat-square) | Core downloader, HTTP server, concurrency, retry logic |
| **Web Frontend** | ![Streamlit](https://img.shields.io/badge/-Streamlit-FF4B4B?logo=streamlit&logoColor=white&style=flat-square) ![Python](https://img.shields.io/badge/-Python-3776AB?logo=python&logoColor=white&style=flat-square) | Browser UI with dark/light mode |
| **API Hosting** | ![Docker](https://img.shields.io/badge/-Docker-2496ED?logo=docker&logoColor=white&style=flat-square) ![HuggingFace](https://img.shields.io/badge/-Hugging%20Face-FFD21E?logo=huggingface&logoColor=black&style=flat-square) | Containerized Go server on HF Spaces |
| **Frontend Hosting** | ![Streamlit Cloud](https://img.shields.io/badge/-Streamlit%20Cloud-FF4B4B?logo=streamlit&logoColor=white&style=flat-square) | Hosted web app |
| **CI/CD** | ![GitHub Actions](https://img.shields.io/badge/-GitHub%20Actions-2088FF?logo=githubactions&logoColor=white&style=flat-square) | Lint, test, build, release |
| **Releases** | ![GoReleaser](https://img.shields.io/badge/-GoReleaser-00ADD8?logo=go&logoColor=white&style=flat-square) | Cross-platform binary builds |

---

## Architecture

```
                          ┌─────────────────────┐
                          │   Streamlit Cloud    │
                          │  gitfetch.streamlit  │
                          │       .app           │
                          └────────┬────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
              (Go backend up?)              (fallback)
                    │                             │
                    v                             v
         ┌──────────────────┐          ┌──────────────────┐
         │  HF Spaces       │          │  Direct GitHub    │
         │  Go REST API     │          │  API calls from   │
         │  /api/v1/...     │          │  Python            │
         └────────┬─────────┘          └──────────────────┘
                  │
      ┌───────────┼───────────┐
      │           │           │
      v           v           v
  ETag Cache   SHA Verify   Retry
  (304 = free) (integrity)  (backoff)
      │
      v
  GitHub REST API v3
```

The Streamlit frontend auto-detects whether the Go backend is available. If it is, requests are routed through Go to benefit from caching, SHA verification, and retries. If not, the frontend calls GitHub directly from Python — users are never blocked.

---

## Key Features

### Backend (Go)

- **Concurrent Downloads** — bounded worker pool via `errgroup` with configurable concurrency
- **SHA-1 Integrity Verification** — hashes files during streaming with `io.TeeReader`, compares against GitHub's SHA
- **ETag Caching** — caches API responses locally; `304 Not Modified` responses are free (don't count against rate limits)
- **Exponential Backoff Retry** — handles 5xx, 429, and network errors with rate-limit-aware delays
- **Streaming I/O** — files stream directly to disk, O(buffer) memory instead of O(file_size)
- **Atomic Writes** — downloads to `.tmp` then renames, preventing partial files
- **Structured Logging** — `log/slog` with debug/info levels via `--verbose`
- **Graceful Shutdown** — signal-aware context propagation (`SIGINT`/`SIGTERM`)
- **HTTP Server Mode** — full REST API with CORS, request-ID, logging, and panic recovery middleware

### Frontend (Streamlit)

- **Dark / Light Mode** — toggle with full GitHub-style theming
- **Go Backend Toggle** — auto-detects and connects to the Go API server
- **File Preview** — browse the file tree with sizes before downloading
- **ZIP Download** — bundles files into a ZIP right in the browser
- **Example URLs** — one-click examples to try immediately
- **SHA Verified Indicator** — shows when files pass integrity checks via the backend

### CLI

- **Dry-run Mode** — preview files and sizes without downloading
- **Custom Output** — `-o` flag for output directory name
- **Timeout Control** — `--timeout` for overall operation deadline
- **Cache Control** — `--no-cache`, `--cache-dir` flags
- **Integrity Toggle** — `--verify` / `--verify=false`

---

## CLI Usage

### Install

```bash
go install github.com/chinmay706/gitf@latest
```

Or download a binary from the [Releases page](https://github.com/chinmay706/Gitfetch/releases).

### Commands

```bash
# Download a folder
gitf download https://github.com/spf13/cobra/tree/main/doc

# Custom output directory
gitf download https://github.com/spf13/cobra/tree/main/doc -o cobra-docs

# Preview files without downloading
gitf download https://github.com/spf13/cobra/tree/main/doc --dry-run

# High concurrency with verbose logging
gitf download <url> -c 20 -v

# Disable SHA verification and caching
gitf download <url> --verify=false --no-cache

# Start the HTTP API server
gitf serve --port 8080 --verbose

# Print version
gitf version
```

### All CLI Flags

| Command | Flag | Default | Description |
|---------|------|---------|-------------|
| `download` | `-o, --output` | `download` | Output directory name |
| `download` | `-c, --concurrency` | `10` | Max parallel downloads |
| `download` | `-t, --timeout` | `5m` | Overall operation timeout |
| `download` | `-v, --verbose` | `false` | Debug-level logging |
| `download` | `--dry-run` | `false` | List files without downloading |
| `download` | `--verify` | `true` | SHA-1 integrity check after download |
| `download` | `--no-cache` | `false` | Disable ETag response caching |
| `download` | `--cache-dir` | `~/.gitf/cache` | Cache directory path |
| `serve` | `--port` | `8080` | Server listen port |
| `serve` | `--cors-origin` | `*` | Allowed CORS origin |
| `serve` | `-v, --verbose` | `false` | Debug-level logging |
| `serve` | `--no-cache` | `false` | Disable caching |
| `serve` | `--cache-dir` | `~/.gitf/cache` | Cache directory path |

---

## API Endpoints

The Go server exposes a REST API at `/api/v1/`:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/health` | Health check — returns `{"status":"ok"}` |
| `GET` | `/api/v1/preview?url=<github-url>` | List files in a GitHub folder (JSON) |
| `POST` | `/api/v1/download` | Download folder as a streamed ZIP |

**Preview example:**

```bash
curl "https://cheesechat-gitfetch-api.hf.space/api/v1/preview?url=https://github.com/spf13/cobra/tree/main/doc"
```

**Download example:**

```bash
curl -X POST https://cheesechat-gitfetch-api.hf.space/api/v1/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://github.com/spf13/cobra/tree/main/doc"}' \
  -o cobra-docs.zip
```

---

## Project Structure

```
gitf/
├── main.go                          # Entry point, signal handling, version injection
├── go.mod / go.sum                  # Go module
├── Dockerfile                       # Multi-stage Docker build for HF Spaces
├── .goreleaser.yml                  # Cross-platform release config
│
├── cmd/                             # CLI layer (Cobra)
│   ├── root.go                      #   Root command + ASCII banner
│   ├── download.go                  #   download subcommand + all flags
│   ├── serve.go                     #   serve subcommand (HTTP server)
│   └── version.go                   #   version subcommand
│
├── internal/
│   ├── downloader/                  # Core download engine
│   │   ├── downloader.go            #   Downloader struct, options, HTTPClient interface
│   │   ├── parser.go                #   GitHub URL parser
│   │   ├── github.go                #   Contents API client, recursive tree walk
│   │   ├── download.go              #   Concurrent download, streaming, SHA verify
│   │   ├── retry.go                 #   Exponential backoff + ETag caching
│   │   ├── errors.go                #   RateLimitError, APIError, IntegrityError
│   │   ├── parser_test.go           #   12-case table-driven URL tests
│   │   ├── download_test.go         #   Unit tests: retry, SHA, cache, context
│   │   └── integration_test.go      #   End-to-end with httptest.Server
│   │
│   ├── server/                      # HTTP API server
│   │   ├── server.go                #   Router, graceful shutdown
│   │   ├── handlers.go              #   /health, /preview, /download handlers
│   │   ├── middleware.go            #   CORS, request-ID, logging, recovery
│   │   └── handlers_test.go         #   8 handler + middleware tests
│   │
│   └── cache/                       # ETag response cache
│       ├── cache.go                 #   File-based cache with TTL
│       └── cache_test.go            #   Put/get/expiry/clear tests
│
├── frontend/                        # Streamlit web UI
│   ├── app.py                       #   Main Streamlit app (dark/light, dual-mode)
│   ├── github_downloader.py         #   Python port of Go downloader
│   ├── api_client.py                #   Go backend API client with fallback
│   ├── requirements.txt             #   streamlit, requests
│   └── .streamlit/config.toml       #   Server config
│
└── .github/workflows/               # CI/CD
    ├── ci.yml                       #   Lint + test + build matrix
    └── release.yml                  #   GoReleaser on tag push
```

---

## Development

### Prerequisites

- **Go 1.22+** (for ServeMux method routing)
- **Python 3.10+** (for Streamlit frontend)
- **Git**

### Build & Test

```bash
git clone https://github.com/chinmay706/Gitfetch.git
cd Gitfetch

# Build
go build -o gitf .

# Run all tests with race detector
go test ./... -race -count=1

# Run the server locally
go run . serve --port 8080 --verbose

# Run the frontend locally
cd frontend
pip install -r requirements.txt
streamlit run app.py
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | GitHub PAT for higher API rate limits (60 → 5,000 req/hr) |
| `GH_TOKEN` | Alternative name (same purpose) |

---

## Deployment

| Component | Platform | Config |
|-----------|----------|--------|
| **Go API** | Hugging Face Spaces (Docker) | `Dockerfile` in repo root, port 7860 |
| **Web UI** | Streamlit Community Cloud | `frontend/app.py`, branch `main` |

The frontend auto-detects the backend. If the HF Space is sleeping (free tier goes idle after ~15 min), the app transparently falls back to direct GitHub API calls.

---

## Live Links

| Service | URL |
|---------|-----|
| Web App | [gitfetch.streamlit.app](https://gitfetch.streamlit.app) |
| API Health | [cheesechat-gitfetch-api.hf.space/api/v1/health](https://cheesechat-gitfetch-api.hf.space/api/v1/health) |
| GitHub Repo | [github.com/chinmay706/Gitfetch](https://github.com/chinmay706/Gitfetch) |

---

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `rate limited (HTTP 403)` | GitHub API limit exceeded | Set `GITHUB_TOKEN` env var or paste token in sidebar |
| `GitHub API error (HTTP 404)` | Repo is private or path doesn't exist | Check URL, ensure repo is public |
| `integrity check failed` | File corrupted during download | Retry; if persistent, use `--verify=false` |
| `Go backend is not reachable` | HF Space is sleeping | Wait ~30s for cold start, or disable the toggle |

---

## Contributing

Contributions are welcome! Please open an issue first to discuss what you'd like to change.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes
4. Push and open a Pull Request

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

Built with [Go](https://go.dev) · [Streamlit](https://streamlit.io) · [Cobra](https://github.com/spf13/cobra)

Hosted on [Hugging Face Spaces](https://huggingface.co/spaces/cheesechat/gitfetch-api) · [Streamlit Cloud](https://gitfetch.streamlit.app)

</div>
