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
import ssl
import urllib.request

import certifi

from common import BUSY, IDLE, OFFLINE, secret

REPO = "synnaxlabs/synnax"
SSL_CONTEXT = ssl.create_default_context(cafile=certifi.where())


def github_token() -> str:
    return secret("GH_TOKEN")


def webhook_secret() -> str:
    """Required, so a misconfigured deploy fails at startup rather than serving a
    listener that accepts unsigned posts."""
    return secret("GH_WEBHOOK_SECRET")


class GitHub:
    """Read-only access to the repository's Actions API."""

    def __init__(self, token: str) -> None:
        self.token = token

    def get(self, path: str) -> dict:
        req = urllib.request.Request(
            f"https://api.github.com/repos/{REPO}{path}",
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        with urllib.request.urlopen(req, timeout=10, context=SSL_CONTEXT) as resp:
            return json.load(resp)

    def runner_status(self) -> tuple[dict[str, int], dict[str, set[str]]]:
        """Maps each runner to its status value, and to the labels it registered
        with, which are what the scaler matches queued jobs against."""
        status: dict[str, int] = {}
        labels: dict[str, set[str]] = {}
        page = 1
        while True:
            runners = self.get(f"/actions/runners?per_page=100&page={page}")["runners"]
            for r in runners:
                if r["status"] != "online":
                    status[r["name"]] = OFFLINE
                else:
                    status[r["name"]] = BUSY if r["busy"] else IDLE
                labels[r["name"]] = {label["name"] for label in r["labels"]}
            if len(runners) < 100:
                return status, labels
            page += 1

    def run_jobs(self, run_id: int) -> list[dict]:
        """Every job of one run, the cheapest way to see step detail for several
        runners at once."""
        return self.get(f"/actions/runs/{run_id}/jobs?per_page=100")["jobs"]

    def unfinished_runs(self) -> list[int]:
        """The runs that can hold a queued job. A run whose hosted jobs already
        started is in progress while its self-hosted jobs still wait."""
        return [
            run["id"]
            for status in ("queued", "in_progress")
            for run in self.get(f"/actions/runs?status={status}&per_page=100")[
                "workflow_runs"
            ]
        ]
