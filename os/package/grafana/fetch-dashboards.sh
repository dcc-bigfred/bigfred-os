#!/usr/bin/env python3
"""Fetch Grafana dashboard JSON files from GitHub repo directories.

Usage: fetch-dashboards.sh <bigfred_ref> <microinit_ref> <out_dir>

Downloads misc/grafana/dashboards/*.json from dcc-bigfred/bigfred and
dcc-bigfred/microinit into <out_dir>/bigfred/ and <out_dir>/microinit/.

Token: BIGFRED_NATIVE_TOKEN / GH_TOKEN / GITHUB_TOKEN (required for private
repos and higher rate limits; public repos work unauthenticated).
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path


API_BASE = os.environ.get("GITHUB_API", "https://api.github.com").rstrip("/")
USER_AGENT = "dcc-bigfred-grafana-dashboards-fetch"

SOURCES = (
    ("dcc-bigfred/bigfred", "misc/grafana/dashboards", "bigfred"),
    ("dcc-bigfred/microinit", "misc/grafana/dashboards", "microinit"),
)


def token() -> str:
    for key in ("BIGFRED_NATIVE_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"):
        value = os.environ.get(key, "").strip()
        if value:
            return value
    return ""


def api_request(url: str) -> bytes:
    headers = {
        "Accept": "application/vnd.github+json",
        "User-Agent": USER_AGENT,
        "X-GitHub-Api-Version": "2022-11-28",
    }
    tok = token()
    if tok:
        headers["Authorization"] = f"Bearer {tok}"
    request = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(request, timeout=120) as response:
        return response.read()


def resolve_ref(repo: str, ref: str) -> str:
    if ref == "latest-release":
        owner, name = repo.split("/", 1)
        payload = json.loads(
            api_request(f"{API_BASE}/repos/{owner}/{name}/releases/latest").decode()
        )
        tag = payload.get("tag_name")
        if not tag:
            raise RuntimeError(f"{repo}: latest release has no tag_name")
        return tag
    if ref.startswith("sha-"):
        return ref[4:]
    return ref


def download_url(url: str) -> bytes:
    headers = {"User-Agent": USER_AGENT}
    tok = token()
    if tok:
        headers["Authorization"] = f"Bearer {tok}"
    request = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(request, timeout=120) as response:
        return response.read()


def fetch_repo_dashboards(repo: str, repo_path: str, ref: str, out_dir: Path) -> int:
    resolved = resolve_ref(repo, ref)
    owner, name = repo.split("/", 1)
    listing_url = (
        f"{API_BASE}/repos/{owner}/{name}/contents/{repo_path}?ref={resolved}"
    )
    payload = json.loads(api_request(listing_url).decode())
    if not isinstance(payload, list):
        raise RuntimeError(f"{repo}@{resolved}: expected directory listing, got object")

    out_dir.mkdir(parents=True, exist_ok=True)
    count = 0
    for entry in payload:
        if not isinstance(entry, dict):
            continue
        if entry.get("type") != "file":
            continue
        filename = entry.get("name", "")
        if not filename.endswith(".json"):
            continue
        dl = entry.get("download_url")
        if not dl:
            raise RuntimeError(f"{repo}/{filename}: missing download_url")
        out_dir.joinpath(filename).write_bytes(download_url(dl))
        count += 1
    if count == 0:
        raise RuntimeError(f"{repo}@{resolved}: no *.json in {repo_path}")
    return count


def main() -> int:
    if len(sys.argv) != 4:
        print(
            f"usage: {sys.argv[0]} <bigfred_ref> <microinit_ref> <out_dir>",
            file=sys.stderr,
        )
        return 2

    bigfred_ref, microinit_ref, out_root = sys.argv[1:]
    refs = (bigfred_ref, microinit_ref)
    out_base = Path(out_root)
    out_base.mkdir(parents=True, exist_ok=True)

    total = 0
    for (repo, repo_path, subdir), ref in zip(SOURCES, refs):
        label = f"{repo}@{ref}"
        try:
            n = fetch_repo_dashboards(repo, repo_path, ref, out_base / subdir)
        except (urllib.error.HTTPError, urllib.error.URLError, RuntimeError) as exc:
            print(f"error: {label}: {exc}", file=sys.stderr)
            return 1
        print(f"OK  {label}: {n} dashboard(s) -> {out_base / subdir}")
        total += n

    print(f"Fetched {total} dashboard(s) into {out_base}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
