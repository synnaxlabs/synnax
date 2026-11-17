#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reruns the failed CI runs of open pull requests in which no job really failed.

    python3 rerun_outage.py <pulls.json>

A GitHub outage fails a run although no job failed: jobs get no runner, or the OK job
sees needs that never ran. The script reruns such a run one time. It reads the open
pull requests from <pulls.json>, a list of objects with number and headRefOid. It
reruns a run only when all of these are true:

- The run failed on its first attempt, so a run is never rerun two times.
- The run has an OK job, and no job other than OK and the job that wakes the shepherd
  failed. A job failed when its conclusion is failure or timed_out, or when it got a
  runner and then was cancelled, which is how a job that reaches its timeout ends.
- The run is the newest CI run for the head of an open pull request.
- The pull request is not in the merge queue.

A failed request for one run does not stop the other runs: the script then exits with
status 1 at the end. Environment: GH_TOKEN and GITHUB_REPOSITORY.
"""

import json
import os
import subprocess
import sys
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

WORKFLOW = "ci.yaml"
GATE = "OK"
# Jobs that run only after a check failed, so their failure is no failed check.
NOT_CHECKS = (GATE, "Wake the shepherd")
FAILED = ("failure", "timed_out")
# A run can wait for runners for more than an hour before GitHub fails it.
CREATED = timedelta(hours=6)
# Limits the job requests for a run that really failed to the sweeps of this time.
COMPLETED = timedelta(hours=2)
QUERY = """query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) { isInMergeQueue headRefOid }
  }
}"""

Api = Callable[..., str]
Run = dict[str, Any]


def _failed(job: dict[str, Any]) -> bool:
    # A job that no runner takes also ends as cancelled, but it has no runner.
    return job["conclusion"] in FAILED or (
        job["conclusion"] == "cancelled" and bool(job["runner_id"])
    )


def outage(run: Run, jobs: list[dict[str, Any]]) -> bool:
    """Returns True if run failed on its first attempt, has an OK job, and no check
    failed."""
    return (
        run["status"] == "completed"
        and run["conclusion"] == "failure"
        and run["run_attempt"] == 1
        and any(j["name"] == GATE for j in jobs)
        and not any(j["name"] not in NOT_CHECKS and _failed(j) for j in jobs)
    )


def candidates(runs: list[Run], heads: set[str], since: datetime) -> list[Run]:
    """Returns the runs worth a look at their jobs: for each head in heads, the newest
    of runs, when it is a first attempt that ended after since. The runs must be in
    order from newest to oldest."""
    seen: set[str] = set()
    out: list[Run] = []
    for run in runs:
        sha = run["head_sha"]
        if sha not in heads or sha in seen:
            continue
        seen.add(sha)
        if (
            run["run_attempt"] == 1
            and datetime.fromisoformat(run["updated_at"]) > since
        ):
            out.append(run)
    return out


def _rows(api: Api, path: str, jq: str) -> list[dict[str, Any]]:
    out = api(path, "--paginate", "--jq", f"{jq} | tojson")
    return [json.loads(line) for line in out.splitlines()]


def _current(api: Api, repo: str, number: int, sha: str) -> bool:
    """Returns True if sha is still the head of the pull request and the pull request
    is not in the merge queue."""
    owner, name = repo.split("/")
    out = api(
        "graphql",
        *("-f", f"query={QUERY}"),
        *("-f", f"owner={owner}", "-f", f"name={name}", "-F", f"number={number}"),
    )
    pull = json.loads(out)["data"]["repository"]["pullRequest"]
    return not pull["isInMergeQueue"] and pull["headRefOid"] == sha


def _rerun(api: Api, repo: str, run: Run, numbers: list[int]) -> bool:
    """Reruns run if it is an outage run and returns True if it did."""
    runs_path = f"repos/{repo}/actions/workflows/{WORKFLOW}/runs"
    run_path = f"repos/{repo}/actions/runs/{run['id']}"
    sha = run["head_sha"]
    jobs = _rows(
        api,
        f"{run_path}/jobs?per_page=100",
        ".jobs[] | {name, conclusion, runner_id}",
    )
    if not outage(run, jobs):
        return False
    # A rerun of an older run cancels the newer run of the same pull request.
    newest = api(
        f"{runs_path}?head_sha={sha}&event=pull_request&per_page=1",
        *("--jq", ".workflow_runs[0].id"),
    )
    if newest.strip() != str(run["id"]):
        return False
    if not all(_current(api, repo, number, sha) for number in numbers):
        return False
    # A rerun of only the failed jobs is not known to start a job that never ran.
    api(f"{run_path}/rerun", "-X", "POST")
    print(f"Reran CI run {run['id']} of #{', #'.join(map(str, numbers))}")
    return True


def sweep(
    api: Api, repo: str, pulls: list[dict[str, Any]], now: datetime
) -> tuple[list[int], list[int]]:
    """Reruns each outage run of pulls. Returns the IDs of the runs it reran and the
    IDs of the runs for which a request failed. The api argument calls `gh api` with
    its arguments and returns the output."""
    heads: dict[str, list[int]] = {}
    for pull in pulls:
        heads.setdefault(pull["headRefOid"], []).append(pull["number"])
    created = (now - CREATED).strftime("%Y-%m-%dT%H:%M:%SZ")
    runs = _rows(
        api,
        f"repos/{repo}/actions/workflows/{WORKFLOW}/runs?status=failure"
        f"&event=pull_request&created=>={created}&per_page=100",
        ".workflow_runs[] | {id, status, conclusion, run_attempt, head_sha,"
        " updated_at}",
    )
    reran: list[int] = []
    failed: list[int] = []
    for run in candidates(runs, set(heads), now - COMPLETED):
        try:
            if _rerun(api, repo, run, heads[run["head_sha"]]):
                reran.append(run["id"])
        except subprocess.CalledProcessError as e:
            print(f"CI run {run['id']}: {e}", file=sys.stderr)
            failed.append(run["id"])
    return reran, failed


def _gh_api(*args: str) -> str:
    return subprocess.run(
        [str(Path(__file__).parent / "gh_api.sh"), *args],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout


def main(argv: list[str]) -> None:
    pulls = json.loads(Path(argv[1]).read_text())
    repo = os.environ["GITHUB_REPOSITORY"]
    _, failed = sweep(_gh_api, repo, pulls, datetime.now(UTC))
    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main(sys.argv)
