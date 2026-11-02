#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Read-only GitHub Actions API access: runner status and the two job reads jobs.py
builds on."""

import json
import ssl
import urllib.request

import certifi

from common import BUSY, IDLE, OFFLINE

REPO = "synnaxlabs/synnax"
SSL_CONTEXT = ssl.create_default_context(cafile=certifi.where())


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

    def pages(self, path: str, *, key: str) -> list[dict]:
        """Every item under key across all pages of a list endpoint."""
        sep = "&" if "?" in path else "?"
        items: list[dict] = []
        page = 1
        while True:
            batch = self.get(f"{path}{sep}per_page=100&page={page}")[key]
            items.extend(batch)
            if len(batch) < 100:
                return items
            page += 1

    def busy(self, runner: str) -> bool:
        """Whether GitHub has a job assigned to the runner now. GitHub sets this at
        assignment, seconds before the runner status poll or a webhook shows it."""
        listed = self.get(f"/actions/runners?name={runner}")["runners"]
        return any(r["busy"] for r in listed)

    def runner_status(self) -> tuple[dict[str, int], dict[str, set[str]]]:
        """Maps each runner to its status value, and to the labels it registered
        with, which are what the scaler matches queued jobs against."""
        status: dict[str, int] = {}
        labels: dict[str, set[str]] = {}
        for r in self.pages("/actions/runners", key="runners"):
            if r["status"] != "online":
                status[r["name"]] = OFFLINE
            else:
                status[r["name"]] = BUSY if r["busy"] else IDLE
            labels[r["name"]] = {label["name"] for label in r["labels"]}
        return status, labels

    def run_jobs(self, run_id: int) -> list[dict]:
        """Every job of one run, the cheapest way to see step detail for several
        runners at once."""
        return self.pages(f"/actions/runs/{run_id}/jobs", key="jobs")

    def unfinished_runs(self) -> list[int]:
        """The runs that can hold a queued job. A run whose hosted jobs already
        started is in progress while its self-hosted jobs still wait."""
        return [
            run["id"]
            for status in ("queued", "in_progress")
            for run in self.pages(f"/actions/runs?status={status}", key="workflow_runs")
        ]
