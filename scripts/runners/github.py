#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Read-only GitHub Actions API access: the credentials, runner status, and the two
job reads jobs.py builds on."""

import json
import os
import ssl
import subprocess
import urllib.request

import certifi

from common import BUSY, IDLE, OFFLINE

REPO = "synnaxlabs/synnax"
SSL_CONTEXT = ssl.create_default_context(cafile=certifi.where())


def github_token() -> str:
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    if token:
        return token
    res = subprocess.run(
        ["gh", "auth", "token"], capture_output=True, text=True, check=False
    )
    if res.returncode != 0:
        raise RuntimeError(f"gh auth token failed: {res.stderr.strip()}")
    return res.stdout.strip()


def webhook_secret() -> str:
    """Required, so a misconfigured deploy fails at startup rather than serving a
    listener that accepts unsigned posts."""
    secret = os.environ.get("GH_WEBHOOK_SECRET")
    if not secret:
        raise RuntimeError("GH_WEBHOOK_SECRET is not set")
    return secret


def gh_get(token: str, path: str) -> dict:
    req = urllib.request.Request(
        f"https://api.github.com/repos/{REPO}{path}",
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urllib.request.urlopen(req, timeout=10, context=SSL_CONTEXT) as resp:
        return json.load(resp)


def runner_status(token: str) -> tuple[dict[str, int], dict[str, set[str]]]:
    """Maps each runner to its status value, and to the labels it registered with,
    which are what the scaler matches queued jobs against."""
    status: dict[str, int] = {}
    labels: dict[str, set[str]] = {}
    page = 1
    while True:
        runners = gh_get(token, f"/actions/runners?per_page=100&page={page}")["runners"]
        for r in runners:
            if r["status"] != "online":
                status[r["name"]] = OFFLINE
            else:
                status[r["name"]] = BUSY if r["busy"] else IDLE
            labels[r["name"]] = {label["name"] for label in r["labels"]}
        if len(runners) < 100:
            return status, labels
        page += 1


def run_jobs(token: str, run_id: int) -> list[dict]:
    """Every job of one run, the cheapest way to see step detail for several runners
    at once."""
    return gh_get(token, f"/actions/runs/{run_id}/jobs?per_page=100")["jobs"]


def queued_runs(token: str) -> list[int]:
    return [
        run["id"]
        for run in gh_get(token, "/actions/runs?status=queued&per_page=100")[
            "workflow_runs"
        ]
    ]
