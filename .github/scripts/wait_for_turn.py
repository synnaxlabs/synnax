#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Holds a merge group's Windows test bot jobs until the groups ahead of it queued
theirs.

A free self-hosted runner takes the oldest queued job, so without this gate a group
whose build ends first takes the test bots from the groups ahead of it. A group goes on
when it is among the first `slots` groups that wait for checks, or when the nearest
group ahead of it that runs the integration suite queued its test bot jobs or never
will.

Usage: wait_for_turn.py <head-sha> <slots>. Reads GH_TOKEN and GITHUB_REPOSITORY.
Exits non-zero when the group leaves the merge queue.
"""

import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

# Must match the job names in ci.yaml and test.integration.yaml.
CI = "CI"
INTEGRATION = "integration"
GATE = "integration / Wait for turn"
TEST_BOT_JOBS = "integration / Test (windows) / "
POLL_SECONDS = 30
MAX_BACKOFF_SECONDS = 300
REQUEST_TIMEOUT_SECONDS = 30
API = "https://api.github.com"
QUEUE_QUERY = """
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    mergeQueue(branch: "main") {
      entries(first: 100) { nodes { position state headCommit { oid } } }
    }
  }
}
"""

# Sends a GET, or a POST when given a JSON body, and returns the decoded response.
Request = Callable[[str, dict[str, Any] | None], Any]


class Transient(Exception):
    """A rate limit or server error that a later poll can retry."""


class Http:
    """Sends authenticated requests to the GitHub API. Raises Transient on a rate
    limit or server error."""

    def __init__(
        self, *, token: str, urlopen: Callable[..., Any] = urllib.request.urlopen
    ) -> None:
        self.token = token
        self.urlopen = urlopen

    def __call__(self, url: str, body: dict[str, Any] | None) -> Any:
        req = urllib.request.Request(
            url,
            data=json.dumps(body).encode() if body is not None else None,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
            },
        )
        try:
            with self.urlopen(req, timeout=REQUEST_TIMEOUT_SECONDS) as resp:
                return json.load(resp)
        except urllib.error.HTTPError as e:
            if e.code in (403, 429) and (
                e.headers.get("Retry-After")
                or e.headers.get("X-RateLimit-Remaining") == "0"
            ):
                raise Transient(f"rate limited ({e.code})") from e
            if e.code >= 500:
                raise Transient(f"server error ({e.code})") from e
            raise
        except (urllib.error.URLError, TimeoutError) as e:
            raise Transient(str(e)) from e


@dataclass(frozen=True, kw_only=True)
class Entry:
    position: int
    state: str
    sha: str


class GitHub:
    """Reads the merge queue and merge group CI jobs."""

    def __init__(self, *, repository: str, request: Request) -> None:
        self.repository = repository
        self.request = request
        self.run_ids: dict[str, int] = {}

    def queue(self) -> list[Entry]:
        owner, name = self.repository.split("/")
        resp = self.request(
            f"{API}/graphql",
            {"query": QUEUE_QUERY, "variables": {"owner": owner, "name": name}},
        )
        if errors := resp.get("errors"):
            if any(e.get("type") == "RATE_LIMITED" for e in errors):
                raise Transient("GraphQL rate limited")
            raise RuntimeError(f"merge queue query failed: {errors}")
        nodes = resp["data"]["repository"]["mergeQueue"]["entries"]["nodes"]
        return [
            Entry(
                position=n["position"],
                state=n["state"],
                sha=(n["headCommit"] or {}).get("oid", ""),
            )
            for n in nodes
        ]

    def jobs(self, sha: str) -> dict[str, dict[str, Any]]:
        """The jobs of the CI run of the merge group at sha, by name. Empty when the
        run does not exist yet."""
        repo = f"{API}/repos/{self.repository}"
        if sha not in self.run_ids:
            query = urllib.parse.urlencode({"head_sha": sha, "event": "merge_group"})
            runs = self.request(f"{repo}/actions/runs?{query}", None)["workflow_runs"]
            ids = [r["id"] for r in runs if r["name"] == CI]
            if not ids:
                return {}
            self.run_ids[sha] = ids[0]
        jobs: dict[str, dict[str, Any]] = {}
        page = 1
        while True:
            url = (
                f"{repo}/actions/runs/{self.run_ids[sha]}/jobs?per_page=100&page={page}"
            )
            batch = self.request(url, None)["jobs"]
            jobs.update((j["name"], j) for j in batch)
            if len(batch) < 100:
                return jobs
            page += 1


def through(jobs: dict[str, dict[str, Any]]) -> bool:
    """True when a group that runs the integration suite queued its test bot jobs or
    never will. A gate that failed or was cancelled counts, because a failed group
    stays in the queue until GitHub removes it."""
    gate = jobs.get(GATE, {})
    if gate.get("status") != "completed":
        return False
    if gate.get("conclusion") != "success":
        return True
    return any(
        name.startswith(TEST_BOT_JOBS)
        and job["status"] in ("queued", "in_progress", "completed")
        for name, job in jobs.items()
    )


def ready(github: GitHub, *, sha: str, slots: int) -> bool:
    """True when the group at sha may queue its test bot jobs. A group's path filter
    sees only its own pull request, so a group that skips the integration suite can
    sit between two that run it: the check skips past such groups."""
    waiting = sorted(
        (e for e in github.queue() if e.state == "AWAITING_CHECKS"),
        key=lambda e: e.position,
    )
    shas = [e.sha for e in waiting]
    if sha not in shas:
        raise SystemExit(
            f"merge group {sha} left the queue ({len(shas)} groups waiting)"
        )
    rank = shas.index(sha) + 1
    print(f"{rank} of {len(shas)} groups waiting for checks, {slots} slots")
    if rank <= slots:
        return True
    for ahead in reversed(shas[: rank - 1]):
        jobs = github.jobs(ahead)
        if jobs.get(INTEGRATION, {}).get("status") != "completed":
            return through(jobs)
    return True


def wait(
    github: GitHub,
    *,
    sha: str,
    slots: int,
    sleep: Callable[[float], None] = time.sleep,
) -> None:
    """Blocks until the group at sha may queue its test bot jobs."""
    delay: float = POLL_SECONDS
    while True:
        try:
            if ready(github, sha=sha, slots=slots):
                return
            delay = POLL_SECONDS
        except Transient as e:
            delay = min(delay * 2, MAX_BACKOFF_SECONDS)
            print(f"{e}, retrying in {delay:.0f} s")
        sleep(delay)


def main() -> None:
    sha, slots = sys.argv[1], int(sys.argv[2])
    github = GitHub(
        repository=os.environ["GITHUB_REPOSITORY"],
        request=Http(token=os.environ["GH_TOKEN"]),
    )
    wait(github, sha=sha, slots=max(slots, 1))


if __name__ == "__main__":
    main()
