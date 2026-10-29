#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the CI jobs that an edit to the CI workflow or the change map selects.

    python3 changed_jobs.py <base>

Compares the workflow and the change map at the base commit with the working tree, and
prints a JSON list of job IDs. A job runs when its block changed or when a filter its
condition reads changed. A job that a selected job needs runs too. An edit to a shared
part runs every job: a top-level key other than jobs, a shared job, or an anchor. A job
added to the needs of OK runs, and that edit is not shared. An empty base selects no
job.
"""

import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

import yaml

WORKFLOW = ".github/workflows/ci.yaml"
FILTERS = ".github/filters.yaml"
SHARED_JOBS = {"changes", "fail-fast", "ok"}
FILTER_REF = re.compile(r"needs\.changes\.outputs\.(\w+)")


class _Loader(yaml.SafeLoader):
    """Keeps the document's anchors, which the composer drops at its end."""

    kept: dict[str, yaml.Node]

    def compose_node(self, parent: yaml.Node | None, index: int) -> yaml.Node | None:
        node = super().compose_node(parent, index)
        if parent is None:
            self.kept = dict(self.anchors)
        return node


def load(text: str) -> tuple[Any, dict[str, Any]]:
    """Returns the document with aliases resolved, and the value of each anchor."""
    loader = _Loader(text)
    try:
        data = loader.get_single_data()
        anchors = {
            k: loader.construct_object(n, deep=True) for k, n in loader.kept.items()
        }
    finally:
        loader.dispose()
    return data, anchors


def paths(value: Any) -> set[str]:
    """Flattens a filter's nested path lists, which aliases make."""
    if isinstance(value, list):
        return set().union(*(paths(v) for v in value))
    return {value}


def changed_filters(old: str, new: str) -> set[str]:
    """Returns the filter keys whose resolved path sets differ."""
    before, after = load(old)[0], load(new)[0]
    return {
        k
        for k in before.keys() | after.keys()
        if paths(before.get(k, [])) != paths(after.get(k, []))
    }


def _needs(jobs: dict[str, Any], job: str) -> list[str]:
    needs = (jobs.get(job) or {}).get("needs", [])
    return [needs] if isinstance(needs, str) else list(needs)


def _shared(jobs: dict[str, Any], job: str) -> Any:
    """Returns the shared part of a job. OK's needs only list the other jobs."""
    block = jobs.get(job) or {}
    return {k: v for k, v in block.items() if not (job == "ok" and k == "needs")}


def select(
    old_ci: str, new_ci: str, *, old_filters: str, new_filters: str
) -> list[str]:
    """Returns the sorted IDs of the jobs in new_ci that the edit selects."""
    (old, old_anchors), (new, new_anchors) = load(old_ci), load(new_ci)
    old_jobs, new_jobs = old.get("jobs", {}), new.get("jobs", {})
    shared = (
        old_anchors != new_anchors
        or {k: v for k, v in old.items() if k != "jobs"}
        != {k: v for k, v in new.items() if k != "jobs"}
        or any(_shared(old_jobs, j) != _shared(new_jobs, j) for j in SHARED_JOBS)
    )
    if shared:
        return sorted(new_jobs)
    filters = changed_filters(old_filters, new_filters)
    selected = {
        j
        for j, block in new_jobs.items()
        if j not in SHARED_JOBS
        and (
            block != old_jobs.get(j)
            or filters & set(FILTER_REF.findall(str(block.get("if", ""))))
        )
    }
    selected |= set(_needs(new_jobs, "ok")) - set(_needs(old_jobs, "ok"))
    pending = list(selected)
    while pending:
        for j in _needs(new_jobs, pending.pop()):
            if j not in selected and j != "changes":
                selected.add(j)
                pending.append(j)
    return sorted(selected)


def show(rev: str, path: str) -> str:
    return subprocess.run(
        ["git", "show", f"{rev}:{path}"], capture_output=True, text=True, check=True
    ).stdout


def main(argv: list[str]) -> None:
    base = argv[1]
    if not base:
        print("[]")
        return
    subprocess.run(
        ["git", "fetch", "--no-tags", "--depth=1", "origin", base], check=True
    )
    jobs = select(
        show(base, WORKFLOW),
        Path(WORKFLOW).read_text(),
        old_filters=show(base, FILTERS),
        new_filters=Path(FILTERS).read_text(),
    )
    print(json.dumps(jobs))


if __name__ == "__main__":
    main(sys.argv)
