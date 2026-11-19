#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""GitHub Actions API access as a GitHub App installation: runner status, runner
removal, and the two job reads jobs.py builds on."""

import json
import ssl
import threading
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from datetime import datetime

import certifi
import jwt

from common import BUSY, IDLE, OFFLINE

API = "https://api.github.com"
REPO = "synnaxlabs/synnax"
SSL_CONTEXT = ssl.create_default_context(cafile=certifi.where())
# Mints a new installation token this many seconds before the current one expires.
REFRESH_MARGIN = 5 * 60

Send = Callable[[urllib.request.Request], dict]


def send(req: urllib.request.Request) -> dict:
    """Sends req to GitHub and returns the decoded JSON response, empty for a response
    with no body."""
    with urllib.request.urlopen(req, timeout=10, context=SSL_CONTEXT) as resp:
        body = resp.read()
    return json.loads(body) if body else {}


class GitHub:
    """Access to the repository's Actions API as the GitHub App app_id, signed
    with its PEM private key. Safe for concurrent use. After GitHub rejects the
    installation token with a 401, the next call mints a new one.

    :param send: Sends a request and returns the decoded JSON response.
    :param clock: The current Unix time in seconds.
    """

    def __init__(
        self,
        app_id: str,
        key: str,
        *,
        send: Send = send,
        clock: Callable[[], float] = time.time,
    ) -> None:
        self.app_id = app_id
        self.key = key
        self.send = send
        self.clock = clock
        self.lock = threading.Lock()
        self.installation: int | None = None
        self.token = ""
        self.expires = 0.0

    def request(self, url: str, *, token: str, method: str = "GET") -> dict:
        return self.send(
            urllib.request.Request(
                url,
                method=method,
                headers={
                    "Authorization": f"Bearer {token}",
                    "Accept": "application/vnd.github+json",
                    "X-GitHub-Api-Version": "2022-11-28",
                },
            )
        )

    def installation_token(self) -> str:
        with self.lock:
            now = self.clock()
            if now < self.expires - REFRESH_MARGIN:
                return self.token
            # iat sits a minute back and exp under GitHub's 10-minute cap, to allow for
            # clock drift.
            app = jwt.encode(
                {"iss": self.app_id, "iat": int(now) - 60, "exp": int(now) + 9 * 60},
                self.key,
                algorithm="RS256",
            )
            if self.installation is None:
                url = f"{API}/repos/{REPO}/installation"
                self.installation = self.request(url, token=app)["id"]
            try:
                minted = self.request(
                    f"{API}/app/installations/{self.installation}/access_tokens",
                    token=app,
                    method="POST",
                )
            except urllib.error.HTTPError as e:
                # The app was installed again under a new ID.
                if e.code == 404:
                    self.installation = None
                raise
            self.token = minted["token"]
            self.expires = datetime.fromisoformat(minted["expires_at"]).timestamp()
            return self.token

    def call(self, path: str, *, method: str = "GET") -> dict:
        token = self.installation_token()
        try:
            return self.request(f"{API}/repos/{REPO}{path}", token=token, method=method)
        except urllib.error.HTTPError as e:
            # GitHub revoked the token early, so the next call mints a new one.
            if e.code == 401:
                with self.lock:
                    if self.token == token:
                        self.expires = 0.0
            raise

    def pages(self, path: str, *, key: str) -> list[dict]:
        """Every item under key across all pages of a list endpoint."""
        sep = "&" if "?" in path else "?"
        items: list[dict] = []
        page = 1
        while True:
            batch = self.call(f"{path}{sep}per_page=100&page={page}")[key]
            items.extend(batch)
            if len(batch) < 100:
                return items
            page += 1

    def offline(self, runner: str) -> bool:
        """Whether GitHub shows the runner offline with no job assigned to it."""
        listed = self.call(f"/actions/runners?name={runner}")["runners"]
        return all(r["status"] == "offline" and not r["busy"] for r in listed)

    def delete(self, runner: str) -> bool:
        """Removes the runner's registration, so GitHub assigns it no more jobs. Returns
        True once no registration is left, and False when GitHub refuses because the
        runner has a job. Needs the app's administration write permission.

        :raises urllib.error.HTTPError: on any other refusal.
        """
        for r in self.call(f"/actions/runners?name={runner}")["runners"]:
            try:
                self.call(f"/actions/runners/{r['id']}", method="DELETE")
            except urllib.error.HTTPError as e:
                if e.code == 422:
                    return False
                raise
        return True

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
