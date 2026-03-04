"""
Thin client for the gitf Go HTTP server.

Falls back gracefully so the Streamlit app can work either against
the Go backend or directly against GitHub.
"""

import io
from typing import Optional

import requests

DEFAULT_SERVER = "http://localhost:8080"
_TIMEOUT = 30


def is_server_up(server: str = DEFAULT_SERVER) -> bool:
    """Check if the Go backend is reachable."""
    try:
        resp = requests.get(f"{server}/api/v1/health", timeout=2)
        return resp.ok and resp.json().get("status") == "ok"
    except Exception:
        return False


def preview(
    github_url: str,
    token: str = "",
    server: str = DEFAULT_SERVER,
) -> dict:
    """Fetch the file list via the Go backend's /preview endpoint.

    Returns a dict with keys: owner, repo, branch, path, files, total_size.
    Each file has: name, path, size, sha.
    """
    params = {"url": github_url}
    if token:
        params["token"] = token

    resp = requests.get(
        f"{server}/api/v1/preview",
        params=params,
        timeout=_TIMEOUT,
    )

    if not resp.ok:
        body = resp.json() if resp.headers.get("content-type", "").startswith("application/json") else {}
        raise RuntimeError(body.get("error", f"Server error (HTTP {resp.status_code})"))

    return resp.json()


def download_zip(
    github_url: str,
    token: str = "",
    server: str = DEFAULT_SERVER,
    on_progress: Optional[callable] = None,
) -> io.BytesIO:
    """Download a GitHub folder as a ZIP via the Go backend.

    Returns an in-memory BytesIO buffer containing the ZIP.
    """
    payload = {"url": github_url}
    if token:
        payload["token"] = token

    resp = requests.post(
        f"{server}/api/v1/download",
        json=payload,
        timeout=120,
        stream=True,
    )

    if not resp.ok:
        try:
            body = resp.json()
            raise RuntimeError(body.get("error", f"Server error (HTTP {resp.status_code})"))
        except ValueError:
            raise RuntimeError(f"Server error (HTTP {resp.status_code})")

    buf = io.BytesIO()
    total = int(resp.headers.get("content-length", 0))
    downloaded = 0

    for chunk in resp.iter_content(chunk_size=8192):
        buf.write(chunk)
        downloaded += len(chunk)
        if on_progress and total > 0:
            on_progress(downloaded, total)

    buf.seek(0)
    return buf
