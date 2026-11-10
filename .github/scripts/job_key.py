#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Keys the CI jobs by their inputs, and finds the jobs whose key passed before.

    python3 job_key.py keys
    python3 job_key.py reused <keys>

keys prints a JSON map from job ID to a hash of the job's inputs: its block in the
workflow, the workflows it calls, the CI actions and scripts, the values of the filters
its inputs read, and the path and blob ID of each file in HEAD that a filter its
condition reads matches. FILTER_OUTPUTS holds the filter outputs as JSON. Only a job
whose condition reads a filter and that no other job needs gets a key.

reused prints the JSON list of jobs whose key has a live pass artifact from a run of
this repository, never a fork. GITHUB_REPOSITORY and GITHUB_REPOSITORY_ID name it.
"""

import hashlib
import json
import os
import re
import subprocess
import sys
from collections.abc import Callable, Iterable
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from typing import Any

from changed_jobs import FILTER_REF, FILTERS, SHARED_JOBS, WORKFLOW, load, paths

# Bump to drop every earlier pass.
VERSION = "1"
# Jobs that run after or around the others, never reused.
UNKEYED = SHARED_JOBS | {"shepherd", "stamp"}
# Every job reads these, so they are part of every key.
COMMON = (".github/actions/**", ".github/scripts/**")
CALLS = re.compile(r"uses:\s*\./(\.github/workflows/[\w.-]+)")
ARTIFACT = "pass-{job}-{key}"

# A file in HEAD as its path and blob ID.
Entry = tuple[str, str]


def glob_regex(glob: str) -> str:
    """Returns a regex for a paths-filter glob. Raises ValueError on a syntax the change
    map does not use."""
    if any(c in glob for c in "{}[]!"):
        raise ValueError(f"unsupported glob syntax: {glob}")
    out, i = [], 0
    while i < len(glob):
        if glob.startswith("**/", i):
            out.append("(?:.*/)?")
            i += 3
        elif glob.startswith("**", i):
            out.append(".*")
            i += 2
        elif glob[i] == "*":
            out.append("[^/]*")
            i += 1
        elif glob[i] == "?":
            out.append("[^/]")
            i += 1
        else:
            out.append(re.escape(glob[i]))
            i += 1
    return "".join(out)


def matcher(globs: Iterable[str]) -> re.Pattern[str]:
    return re.compile("|".join(f"(?:{glob_regex(g)})" for g in sorted(globs)))


def tree(rev: str = "HEAD") -> list[Entry]:
    """Returns every file in rev, sorted by path."""
    out = subprocess.run(
        ["git", "ls-tree", "-r", "-z", "--full-tree", rev],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    entries = []
    for line in out.split("\0"):
        if line:
            meta, path = line.split("\t", 1)
            entries.append((path, meta.split()[2]))
    return sorted(entries)


def _refs(value: Any, names: Iterable[str]) -> list[str]:
    """Returns the sorted filter names that value reads."""
    return sorted(set(FILTER_REF.findall(json.dumps(value))) & set(names))


def _needed(jobs: dict[str, Any]) -> set[str]:
    """Returns the jobs whose outputs another keyed job reads."""
    needed: set[str] = set()
    for job, block in jobs.items():
        if job in UNKEYED:
            continue
        needs = (block or {}).get("needs", [])
        needed |= {needs} if isinstance(needs, str) else set(needs)
    return needed


def reusable(jobs: dict[str, Any], filters: Iterable[str]) -> dict[str, list[str]]:
    """Returns each job that can reuse a pass, with the filters its condition reads."""
    needed = _needed(jobs)
    out = {}
    for job, block in jobs.items():
        if job in UNKEYED or job in needed:
            continue
        refs = _refs((block or {}).get("if", ""), filters)
        if refs:
            out[job] = refs
    return out


def called(workflow: str, read: Callable[[str], str]) -> set[str]:
    """Returns workflow and every workflow it calls, directly or not."""
    seen, pending = set(), [workflow]
    while pending:
        path = pending.pop()
        if path not in seen:
            seen.add(path)
            pending.extend(CALLS.findall(read(path)))
    return seen


def digest(lines: Iterable[str]) -> str:
    h = hashlib.sha256()
    for line in lines:
        h.update(line.encode())
        h.update(b"\n")
    return h.hexdigest()


def keys(
    ci: str,
    filters: str,
    entries: list[Entry],
    *,
    read: Callable[[str], str],
    outputs: dict[str, str],
) -> dict[str, str]:
    """Returns the key of each job that can reuse a pass."""
    jobs = load(ci)[0].get("jobs", {})
    globs = {k: paths(v) for k, v in load(filters)[0].items()}
    selected = reusable(jobs, globs)
    files: dict[str, str] = {}

    def of(name: str, patterns: Iterable[str]) -> str:
        if name not in files:
            pattern = matcher(patterns)
            files[name] = digest(f"{p} {b}" for p, b in entries if pattern.fullmatch(p))
        return files[name]

    common = of("", COMMON)
    out = {}
    for job, refs in selected.items():
        block = jobs[job]
        uses = block.get("uses", "")
        workflows = called(uses[2:], read) if uses.startswith("./") else set()
        lines = [
            VERSION,
            job,
            json.dumps(block, sort_keys=True, default=str),
            common,
            *(f"{f}={outputs[f]}" for f in _refs(block.get("with", {}), globs)),
            *(f"{w} {of(w, [w])}" for w in sorted(workflows)),
            *(f"{r} {of(r, globs[r])}" for r in refs),
        ]
        out[job] = digest(lines)[:40]
    return out


def passed(artifacts: list[dict[str, Any]], repository_id: int) -> bool:
    """Returns whether a live artifact comes from a run of this repository."""
    return any(
        not a["expired"] and a["workflow_run"]["head_repository_id"] == repository_id
        for a in artifacts
    )


def reused(
    job_keys: dict[str, str],
    *,
    fetch: Callable[[str], list[dict[str, Any]]],
    repository_id: int,
) -> list[str]:
    """Returns the sorted jobs whose key has a pass artifact."""
    names = {j: ARTIFACT.format(job=j, key=k) for j, k in job_keys.items()}
    with ThreadPoolExecutor(max_workers=8) as pool:
        found = dict(zip(names, pool.map(fetch, names.values()), strict=True))
    return sorted(
        j for j, artifacts in found.items() if passed(artifacts, repository_id)
    )


def fetch_artifacts(repository: str) -> Callable[[str], list[dict[str, Any]]]:
    script = Path(__file__).with_name("gh_api.sh")

    def fetch(name: str) -> list[dict[str, Any]]:
        out = subprocess.run(
            [
                str(script),
                f"repos/{repository}/actions/artifacts?name={name}&per_page=20",
            ],
            capture_output=True,
            text=True,
            check=True,
        ).stdout
        artifacts: list[dict[str, Any]] = json.loads(out)["artifacts"]
        return artifacts

    return fetch


def main(argv: list[str]) -> None:
    if argv[1] == "keys":
        result: Any = keys(
            Path(WORKFLOW).read_text(),
            Path(FILTERS).read_text(),
            tree(),
            read=lambda p: Path(p).read_text(),
            outputs=json.loads(os.environ["FILTER_OUTPUTS"]),
        )
    elif argv[1] == "reused":
        result = reused(
            json.loads(argv[2]),
            fetch=fetch_artifacts(os.environ["GITHUB_REPOSITORY"]),
            repository_id=int(os.environ["GITHUB_REPOSITORY_ID"]),
        )
    else:
        raise ValueError(f"unknown command: {argv[1]}")
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main(sys.argv)
