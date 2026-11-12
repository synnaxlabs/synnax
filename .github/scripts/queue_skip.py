#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the change outputs of a CI run, less the jobs a merge group can skip.

    python3 queue_skip.py

Prints one key=value line per filter, and a ci line with the JSON list of the jobs
that an edit to CI selects. Every event other than merge_group gets its inputs back.

A merge group skips a job that passed on its pull request when both tested the same
tree: the group and the pull request head have one tree, and the head contains the
base of the group. A pull request run tests the merge of the head into its base, so
the second condition makes that merge equal to the head. It is false for a group
that sits behind another group. The proof is a successful pull_request run of the CI
workflow file on that head. A job whose definition reads the event is not the same
job in the two runs, so it runs. A job that still runs goes in the ci list, and each
filter becomes false. No proof, or a GitHub API failure, changes nothing.

Reads TOUCHED, the JSON outputs of the path filter, and CI_JOBS, the output of
changed_jobs.py. Reads GH_TOKEN, GITHUB_EVENT_NAME, GITHUB_EVENT_PATH, and
GITHUB_REPOSITORY.
"""

import json
import os
import re
import sys
import urllib.error
import urllib.parse
from pathlib import Path
from typing import Any

from changed_jobs import FILTER_REF, WORKFLOW, load
from wait_for_turn import API, Http, Request, Transient

EVENT_REF = re.compile(r"github\.event")
GROUP_REF = re.compile(r"/pr-(\d+)-[0-9a-f]{40}$")


class Unproven(Exception):
    """The pull request did not test the tree of the merge group."""


def _pages(request: Request, url: str, key: str, **params: str) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    page = 1
    while True:
        query = urllib.parse.urlencode({**params, "per_page": 100, "page": page})
        batch = request(f"{url}?{query}", None)[key]
        items += batch
        if len(batch) < 100:
            return items
        page += 1


def conclusions(
    group: dict[str, Any], *, repository: str, request: Request
) -> dict[str, str]:
    """Returns the conclusion of each job of the pull request run that tested the tree
    of the merge group, by name. Raises Unproven when no such run exists."""
    match = GROUP_REF.search(group["head_ref"])
    if match is None:
        raise Unproven(f"{group['head_ref']} names no pull request")
    repo = f"{API}/repos/{repository}"
    head = request(f"{repo}/pulls/{match[1]}", None)["head"]["sha"]
    print(f"Pull request #{match[1]} at {head}", file=sys.stderr)
    trees = [
        request(f"{repo}/git/commits/{sha}", None)["tree"]["sha"]
        for sha in (group["head_sha"], head)
    ]
    if trees[0] != trees[1]:
        raise Unproven(f"the group has tree {trees[0]}, and the head {trees[1]}")
    compare = request(f"{repo}/compare/{group['base_sha']}...{head}", None)
    if compare["behind_by"] != 0:
        raise Unproven(f"the head is {compare['behind_by']} commits behind the base")
    runs = [
        r
        for r in _pages(
            request,
            f"{repo}/actions/workflows/ci.yaml/runs",
            "workflow_runs",
            head_sha=head,
            event="pull_request",
            status="success",
        )
        if r["head_sha"] == head
        and r["event"] == "pull_request"
        and r["path"] == WORKFLOW
        and r["repository"]["full_name"] == repository
        and r["status"] == "completed"
        and r["conclusion"] == "success"
    ]
    if not runs:
        raise Unproven(f"no pull request run of {WORKFLOW} passed on {head}")
    print(f"Run {runs[0]['id']} passed on tree {trees[0]}", file=sys.stderr)
    jobs = _pages(request, f"{repo}/actions/runs/{runs[0]['id']}/jobs", "jobs")
    return {j["name"]: j["conclusion"] for j in jobs}


def passed(name: str, conclusions: dict[str, str]) -> bool:
    """Returns True if the job passed. A call to a workflow passed when one of its jobs
    passed and each other one was skipped: the call inputs skip a job, and they are the
    same in the merge group."""
    parts = [
        c for n, c in conclusions.items() if n == name or n.startswith(f"{name} / ")
    ]
    return "success" in parts and all(c in ("success", "skipped") for c in parts)


def _filters(value: Any) -> set[str]:
    return set(FILTER_REF.findall(str(value)))


def skip(
    filters: dict[str, str],
    ci: list[str],
    *,
    jobs: dict[str, Any],
    conclusions: dict[str, str],
) -> tuple[dict[str, str], list[str]]:
    """Returns the filters and the ci list with the jobs that passed left out. A job
    that a remaining job needs runs too. A filter that a remaining job reads outside
    its condition keeps its value."""
    selected = {
        j
        for j, block in jobs.items()
        if j in ci or any(filters.get(f) == "true" for f in _filters(block.get("if")))
    }
    proven = {
        j
        for j in selected
        if not EVENT_REF.search(str(jobs[j]))
        and passed(jobs[j].get("name", j), conclusions)
    }
    if not proven:
        return filters, ci
    pending = list(selected - proven)
    while pending:
        needs = jobs[pending.pop()].get("needs", [])
        for j in {needs} if isinstance(needs, str) else set(needs):
            if j in proven:
                proven.remove(j)
                pending.append(j)
    remaining = selected - proven
    pinned = set().union(
        *(_filters({k: v for k, v in jobs[j].items() if k != "if"}) for j in remaining)
    )
    print(f"Skipped: {' '.join(sorted(proven))}", file=sys.stderr)
    print(f"Still run: {' '.join(sorted(remaining))}", file=sys.stderr)
    return (
        {f: str(v == "true" and f in pinned).lower() for f, v in filters.items()},
        sorted(remaining),
    )


def main(request: Request | None = None) -> None:
    touched = json.loads(os.environ["TOUCHED"])
    filters = {k: v for k, v in touched.items() if v in ("true", "false")}
    ci = json.loads(os.environ["CI_JOBS"])
    if os.environ["GITHUB_EVENT_NAME"] == "merge_group":
        group = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
        try:
            found = conclusions(
                group["merge_group"],
                repository=os.environ["GITHUB_REPOSITORY"],
                request=request or Http(token=os.environ["GH_TOKEN"]),
            )
        except Unproven as e:
            print(f"Every job runs: {e}.", file=sys.stderr)
        except (Transient, urllib.error.HTTPError) as e:
            print(
                f"::warning::Every job runs: the GitHub API failed: {e}",
                file=sys.stderr,
            )
        else:
            jobs = load(Path(WORKFLOW).read_text())[0]["jobs"]
            filters, ci = skip(filters, ci, jobs=jobs, conclusions=found)
    for key, value in filters.items():
        print(f"{key}={value}")
    print(f"ci={json.dumps(ci)}")


if __name__ == "__main__":
    main()
