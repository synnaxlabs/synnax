#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fails when a repository webhook stops reaching its receiver.

Reads the hook's deliveries from the last hour, newest first, up to MAX_DELIVERIES.
Fails when at least half of them got no 2xx response, or when there were none while
other workflow runs started.

Usage: hook_health.py <hook-id>. Reads GH_TOKEN, GITHUB_REPOSITORY, and GITHUB_RUN_ID.
"""

import json
import os
import re
import sys
import urllib.parse
import urllib.request
from collections import Counter
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from typing import Any

API = "https://api.github.com"
WINDOW = timedelta(hours=1)
# A busy hour sends thousands of deliveries. The newest thousand are enough to judge.
MAX_DELIVERIES = 1000
FAILURE_RATIO = 0.5
REQUEST_TIMEOUT_SECONDS = 30

# Sends a GET and returns the decoded body and the URL of the next page, if any.
Fetch = Callable[[str], tuple[Any, str | None]]


class Http:
    """Sends authenticated GET requests to the GitHub API."""

    def __init__(
        self, *, token: str, urlopen: Callable[..., Any] = urllib.request.urlopen
    ) -> None:
        self.token = token
        self.urlopen = urlopen

    def __call__(self, url: str) -> tuple[Any, str | None]:
        req = urllib.request.Request(
            url,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
            },
        )
        with self.urlopen(req, timeout=REQUEST_TIMEOUT_SECONDS) as resp:
            link = re.search(r'<([^>]+)>;\s*rel="next"', resp.headers.get("Link", ""))
            return json.load(resp), link.group(1) if link else None


def deliveries(
    fetch: Fetch, *, repository: str, hook: int, since: datetime
) -> list[dict[str, Any]]:
    """The hook's deliveries at or after since, newest first, at most
    MAX_DELIVERIES."""
    url: str | None = f"{API}/repos/{repository}/hooks/{hook}/deliveries?per_page=100"
    found: list[dict[str, Any]] = []
    while url is not None and len(found) < MAX_DELIVERIES:
        page, url = fetch(url)
        recent = [d for d in page if datetime.fromisoformat(d["delivered_at"]) >= since]
        found.extend(recent)
        if len(recent) < len(page):
            break
    return found[:MAX_DELIVERIES]


def other_runs_started(
    fetch: Fetch, *, repository: str, since: datetime, run_id: int
) -> bool:
    """True when a workflow run other than run_id started at or after since."""
    query = urllib.parse.urlencode(
        {"created": f">={since:%Y-%m-%dT%H:%M:%SZ}", "per_page": 2}
    )
    body, _ = fetch(f"{API}/repos/{repository}/actions/runs?{query}")
    return any(r["id"] != run_id for r in body["workflow_runs"])


def check(
    fetch: Fetch, *, repository: str, hook: int, run_id: int, now: datetime
) -> tuple[bool, str]:
    """Returns whether the hook is healthy and a one-line summary."""
    since = now - WINDOW
    found = deliveries(fetch, repository=repository, hook=hook, since=since)
    if not found:
        if other_runs_started(fetch, repository=repository, since=since, run_id=run_id):
            return False, "no deliveries in the last hour while workflow runs started"
        return True, "no deliveries and no workflow runs in the last hour"
    failed = Counter(d["status_code"] for d in found if d["status_code"] // 100 != 2)
    total = sum(failed.values())
    summary = f"{total} of {len(found)} deliveries in the last hour failed"
    if failed:
        codes = ", ".join(f"{n} x {code}" for code, n in failed.most_common())
        summary += f" ({codes})"
    return total < FAILURE_RATIO * len(found), summary


def main() -> None:
    healthy, summary = check(
        Http(token=os.environ["GH_TOKEN"]),
        repository=os.environ["GITHUB_REPOSITORY"],
        hook=int(sys.argv[1]),
        run_id=int(os.environ["GITHUB_RUN_ID"]),
        now=datetime.now(UTC),
    )
    print(summary)
    if not healthy:
        sys.exit(1)


if __name__ == "__main__":
    main()
