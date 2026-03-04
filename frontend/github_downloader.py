"""
Pure-Python port of the Go gitf downloader.
Provides URL parsing, recursive file listing, and in-memory zip creation
via the GitHub REST API.
"""

import io
import zipfile
from urllib.parse import urlparse

import requests

API_BASE = "https://api.github.com"
MAX_FILES = 500
MAX_SIZE_WARN = 100 * 1024 * 1024  # 100 MB


class GitfError(Exception):
    """Base error for gitf operations."""


class RateLimitError(GitfError):
    """Raised when the GitHub API rate-limits a request."""

    def __init__(self, reset_at: str = ""):
        msg = "GitHub API rate limit exceeded."
        if reset_at:
            msg += f" Resets at {reset_at}."
        msg += " Add a GitHub token in the sidebar to increase limits."
        super().__init__(msg)


class APIError(GitfError):
    """Raised for non-success GitHub API responses."""

    def __init__(self, status_code: int, body: str = ""):
        msg = f"GitHub API error (HTTP {status_code})"
        if body:
            msg += f": {body[:300]}"
        super().__init__(msg)
        self.status_code = status_code


# ---------------------------------------------------------------------------
# URL parsing (mirrors internal/downloader/parser.go)
# ---------------------------------------------------------------------------

def parse_github_url(raw_url: str) -> dict:
    """Parse a GitHub folder URL into its components.

    Returns dict with keys: owner, repo, branch, path.
    Raises GitfError on invalid input.
    """
    parsed = urlparse(raw_url.strip())

    if parsed.hostname != "github.com":
        raise GitfError("URL must be a github.com link.")

    parts = [p for p in parsed.path.split("/") if p]

    if len(parts) < 4 or parts[2] != "tree":
        raise GitfError(
            "Invalid GitHub folder URL. "
            "Expected format: https://github.com/owner/repo/tree/branch/path"
        )

    return {
        "owner": parts[0],
        "repo": parts[1],
        "branch": parts[3],
        "path": "/".join(parts[4:]),
    }


# ---------------------------------------------------------------------------
# GitHub API helpers
# ---------------------------------------------------------------------------

def _session(token: str = "") -> requests.Session:
    s = requests.Session()
    s.headers.update({
        "User-Agent": "gitf-web",
        "Accept": "application/vnd.github.v3+json",
    })
    if token:
        s.headers["Authorization"] = f"token {token}"
    return s


def _check_response(resp: requests.Response) -> None:
    if resp.ok:
        return

    if resp.status_code in (403, 429):
        remaining = resp.headers.get("X-RateLimit-Remaining", "")
        if remaining == "0" or "rate limit" in resp.text.lower():
            reset = resp.headers.get("X-RateLimit-Reset", "")
            raise RateLimitError(reset_at=reset)

    raise APIError(resp.status_code, resp.text.strip())


# ---------------------------------------------------------------------------
# File collection (mirrors CollectFiles / collectFilesRecursive)
# ---------------------------------------------------------------------------

def collect_files(
    info: dict,
    token: str = "",
    on_status=None,
) -> list[dict]:
    """Recursively list all files under a GitHub folder.

    Returns a list of dicts with keys: name, path, size, download_url, api_url.
    Calls on_status(message) if provided to report progress.
    """
    sess = _session(token)
    api_url = (
        f"{API_BASE}/repos/{info['owner']}/{info['repo']}"
        f"/contents/{info['path']}?ref={info['branch']}"
    )

    files: list[dict] = []
    _collect_recursive(sess, api_url, files, on_status)
    return files


def _collect_recursive(
    sess: requests.Session,
    api_url: str,
    files: list[dict],
    on_status,
) -> None:
    if len(files) >= MAX_FILES:
        return

    resp = sess.get(api_url, timeout=30)
    _check_response(resp)
    contents = resp.json()

    for item in contents:
        if len(files) >= MAX_FILES:
            return
        if item["type"] == "file":
            files.append({
                "name": item["name"],
                "path": item["path"],
                "size": item.get("size", 0),
                "download_url": item["download_url"],
                "api_url": item.get("url", ""),
            })
            if on_status:
                on_status(f"Found {len(files)} files...")
        elif item["type"] == "dir":
            _collect_recursive(sess, item["url"], files, on_status)


# ---------------------------------------------------------------------------
# Zip creation (mirrors downloadFiles + streamFile)
# ---------------------------------------------------------------------------

def download_as_zip(
    files: list[dict],
    base_prefix: str,
    token: str = "",
    on_progress=None,
) -> io.BytesIO:
    """Download files and return an in-memory zip.

    on_progress(completed, total) is called after each file.
    """
    sess = _session(token)
    buf = io.BytesIO()
    total = len(files)

    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        for i, f in enumerate(files):
            rel = f["path"]
            if base_prefix and rel.startswith(base_prefix):
                rel = rel[len(base_prefix):]
            rel = rel.lstrip("/")

            resp = sess.get(f["download_url"], timeout=60)
            _check_response(resp)
            zf.writestr(rel, resp.content)

            if on_progress:
                on_progress(i + 1, total)

    buf.seek(0)
    return buf


def human_size(b: int) -> str:
    for unit in ("B", "KB", "MB", "GB"):
        if abs(b) < 1024:
            return f"{b:.1f} {unit}" if unit != "B" else f"{b} {unit}"
        b /= 1024
    return f"{b:.1f} TB"
