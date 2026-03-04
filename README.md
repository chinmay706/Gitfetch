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

[**Try the Web App**](https://gitfetch.streamlit.app) · [**API**](https://cheesechat-gitfetch-api.hf.space/api/v1/health) · [**CLI Install**](#cli)

</div>

---

## How to Use

| Interface | For whom | Link |
|-----------|----------|------|
| **Web App** | Anyone | [gitfetch.streamlit.app](https://gitfetch.streamlit.app) |
| **CLI** | Developers | `go install github.com/chinmay706/gitf@latest` |
| **REST API** | Scripts / services | `GET /api/v1/preview?url=...` |

---

## Tech Stack

![Go](https://img.shields.io/badge/-Go-00ADD8?logo=go&logoColor=white&style=flat-square)
![Python](https://img.shields.io/badge/-Python-3776AB?logo=python&logoColor=white&style=flat-square)
![Streamlit](https://img.shields.io/badge/-Streamlit-FF4B4B?logo=streamlit&logoColor=white&style=flat-square)
![Docker](https://img.shields.io/badge/-Docker-2496ED?logo=docker&logoColor=white&style=flat-square)
![GitHub Actions](https://img.shields.io/badge/-GitHub%20Actions-2088FF?logo=githubactions&logoColor=white&style=flat-square)
![Hugging Face](https://img.shields.io/badge/-Hugging%20Face-FFD21E?logo=huggingface&logoColor=black&style=flat-square)

---

## Features

- **Concurrent downloads** with bounded worker pool (`errgroup`)
- **SHA-1 integrity verification** — hashes during stream, zero extra I/O
- **ETag caching** — `304 Not Modified` responses are free (no rate limit cost)
- **Exponential backoff retry** with rate-limit-aware delays
- **Streaming I/O** — files go straight to disk, not memory
- **Atomic writes** — `.tmp` then rename, no partial files
- **HTTP server mode** with CORS, request-ID, logging, and recovery middleware
- **Dark / light mode** in the web UI with GitHub-style theming
- **Auto-fallback** — web app works with or without the Go backend

---

## CLI

```bash
go install github.com/chinmay706/gitf@latest

# Download a folder
gitf download https://github.com/spf13/cobra/tree/main/doc

# Custom output + dry run
gitf download <url> -o my-folder --dry-run

# Start the API server
gitf serve --port 8080 --verbose
```

Key flags: `-o` output dir, `-c` concurrency, `-t` timeout, `-v` verbose, `--dry-run`, `--verify=false`, `--no-cache`

---

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/health` | Health check |
| `GET` | `/api/v1/preview?url=<github-url>` | List files (JSON) |
| `POST` | `/api/v1/download` | Download as streamed ZIP |

```bash
curl "https://cheesechat-gitfetch-api.hf.space/api/v1/preview?url=https://github.com/spf13/cobra/tree/main/doc"
```

---

## Development

```bash
git clone https://github.com/chinmay706/Gitfetch.git && cd Gitfetch

go build -o gitf .          # build
go test ./... -race          # test

cd frontend                  # run web UI locally
pip install -r requirements.txt
streamlit run app.py
```

Set `GITHUB_TOKEN` env var for higher API rate limits (60 -> 5,000 req/hr).

---

## Deployment

| Component | Platform | URL |
|-----------|----------|-----|
| Go API | Hugging Face Spaces (Docker) | [cheesechat-gitfetch-api.hf.space](https://cheesechat-gitfetch-api.hf.space/api/v1/health) |
| Web UI | Streamlit Cloud | [gitfetch.streamlit.app](https://gitfetch.streamlit.app) |

---

## License

[MIT](LICENSE)
