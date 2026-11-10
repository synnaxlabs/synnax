#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Keys the CI jobs by their inputs and finds the keys that passed before.

    python3 job_key.py keys
    python3 job_key.py reused <keys>
    python3 job_key.py changes <pull request> <head>

A key hashes the job's block, the workflows it calls, and the files its filters match.
Only its own module counts unit tests. A test workflow counts no prose. Go and Python
files count without comments where the job reads none. A lint job also gets a code key,
whose pass runs only its lint. The Go jobs key HEAD_SHA, the pull request head they
test. changes prints the Go files each Go job's inputs changed since its last pass on
the pull request, or all.
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
from typing import Any, NamedTuple

import comments
from changed_jobs import FILTER_REF, FILTERS, SHARED_JOBS, WORKFLOW, load, paths

# Bump to drop every earlier pass.
VERSION = "2"
UNIT_TESTS = "unit_tests"
# Jobs that run around the others.
UNKEYED = SHARED_JOBS | {"shepherd", "stamp"}
# Every job reads these.
COMMON = (".github/actions/**", ".github/scripts/**")
CALLS = re.compile(r"uses:\s*\./(\.github/workflows/[\w.-]+)")
ARTIFACT = "pass-{job}-{key}"
# Which unit tests a filter counts. Any other value names a filter.
ALL, NONE = "", "-"
# No test workflow reads prose.
PROSE = ".md"
# The Go and Python comments each workflow reads. Lint reads its own in lint only, code
# reads none, and any other job reads all.
LINT, CODE = "lint", "code"
MODES = {
    "./.github/workflows/test.cpp.yaml": CODE,
    "./.github/workflows/test.go.yaml": LINT,
    "./.github/workflows/test.integration.yaml": CODE,
    "./.github/workflows/test.py.yaml": LINT,
    "./.github/workflows/test.ts.yaml": CODE,
}

# The Go jobs test the head of a pull request, not its merge commit.
GO = "./.github/workflows/test.go.yaml"

# A file as its path and blob ID.
Entry = tuple[str, str]


class Snapshot(NamedTuple):
    """The files of a commit, and a reader of their content by path."""

    entries: list[Entry]
    content: Callable[[str], bytes]


def glob_regex(glob: str) -> str:
    """Returns a regex for a change map glob, or raises ValueError on other syntax."""
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


def snapshot(rev: str) -> Snapshot:
    """Returns the files of rev, read from the object store."""
    entries = tree(rev)
    blobs = dict(entries)
    batch = subprocess.Popen(
        ["git", "cat-file", "--batch"], stdin=subprocess.PIPE, stdout=subprocess.PIPE
    )
    stdin, stdout = batch.stdin, batch.stdout
    assert stdin is not None
    assert stdout is not None

    def content(path: str) -> bytes:
        stdin.write(f"{blobs[path]}\n".encode())
        stdin.flush()
        size = int(stdout.readline().split()[2])
        data = stdout.read(size)
        stdout.read(1)
        return data

    return Snapshot(entries, content)


def _refs(value: Any, names: Iterable[str]) -> list[str]:
    """Returns the filter names that value reads, in the order it first reads them."""
    names = set(names)
    return [
        r for r in dict.fromkeys(FILTER_REF.findall(json.dumps(value))) if r in names
    ]


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
    """Returns each keyed job with the filters its condition reads, its own first."""
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
    files: Snapshot,
    *,
    read: Callable[[str], str],
    outputs: dict[str, str],
    head: Snapshot | None = None,
) -> dict[str, dict[str, str]]:
    """Returns the keys of each job that can reuse a pass. A Go job keys head when it
    is given."""
    jobs = load(ci)[0].get("jobs", {})
    globs = {k: paths(v) for k, v in load(filters)[0].items()}
    tests = matcher(globs[UNIT_TESTS])

    def hasher(snap: Snapshot) -> Callable[..., str]:
        stripped: dict[str, str] = {}
        cache: dict[tuple[str, str, bool, bool], str] = {}

        def version(path: str, blob: str, code: bool) -> str:
            if (
                not code
                or not path.endswith(comments.SUFFIXES)
                or tests.fullmatch(path)
            ):
                return blob
            if path not in stripped:
                text = comments.strip(path, snap.content(path))
                stripped[path] = blob if text is None else digest([text])
            return stripped[path]

        def of(
            name: str, patterns: Iterable[str], *, own: str, code: bool, prose: bool
        ) -> str:
            if (name, own, code, prose) not in cache:
                pattern, mine = matcher(patterns), matcher(owned[own])
                cache[name, own, code, prose] = digest(
                    f"{p} {version(p, b, code)}"
                    for p, b in snap.entries
                    if pattern.fullmatch(p)
                    and (not tests.fullmatch(p) or mine.fullmatch(p))
                    and (prose or not p.endswith(PROSE))
                )
            return cache[name, own, code, prose]

        return of

    refs = reusable(jobs, globs)
    owners = {r[0] for r in refs.values()}
    # A test workflow runs the unit tests of its own paths only.
    owned: dict[str, list[str]] = {ALL: ["**"], NONE: []}
    for name in owners:
        mine = set(globs[name])
        owned[name] = sorted(
            mine.difference(*(set(globs[k]) for k in owners if set(globs[k]) < mine))
        )

    base = hasher(files)
    heads = hasher(head) if head else base

    def key(job: str, refs: list[str], *, own_code: bool, dep_code: bool) -> str:
        block = jobs[job]
        uses = block.get("uses", "")
        of = heads if uses == GO else base
        workflows = called(uses[2:], read) if uses.startswith("./") else set()
        own, deps = refs[0], refs[1:]
        tested, prose = (own, False) if uses in MODES else (ALL, True)
        return digest(
            [
                VERSION,
                job,
                f"{own_code} {dep_code}",
                json.dumps(block, sort_keys=True, default=str),
                of("", COMMON, own=ALL, code=False, prose=True),
                *(f"{f}={outputs[f]}" for f in _refs(block.get("with", {}), globs)),
                *(
                    f"{w} {base(w, [w], own=ALL, code=False, prose=True)}"
                    for w in sorted(workflows)
                ),
                f"{own} {of(own, globs[own], own=tested, code=own_code, prose=prose)}",
                *(
                    f"{r} {of(r, globs[r], own=NONE, code=dep_code, prose=prose)}"
                    for r in deps
                ),
            ]
        )[:40]

    out = {}
    for job, inputs in refs.items():
        mode = MODES.get(jobs[job].get("uses", ""))
        if mode == LINT:
            out[job] = {
                "key": key(job, inputs, own_code=False, dep_code=True),
                "code": key(job, inputs, own_code=True, dep_code=True),
            }
        else:
            code = mode == CODE
            out[job] = {"key": key(job, inputs, own_code=code, dep_code=code)}
    return out


def passed(artifacts: list[dict[str, Any]], repository_id: int) -> bool:
    """Returns whether a live artifact comes from a run of this repository."""
    return bool(_live(artifacts, repository_id))


def reused(
    job_keys: dict[str, dict[str, str]],
    *,
    fetch: Callable[[str], list[dict[str, Any]]],
    repository_id: int,
) -> dict[str, list[str]]:
    """Returns the jobs whose key passed, and the others whose code key passed."""
    names = {
        (j, kind): ARTIFACT.format(job=j, key=k)
        for j, kinds in job_keys.items()
        for kind, k in kinds.items()
    }
    with ThreadPoolExecutor(max_workers=8) as pool:
        found = dict(zip(names, pool.map(fetch, names.values()), strict=True))
    hit = {
        name for name, artifacts in found.items() if passed(artifacts, repository_id)
    }
    skip = sorted(j for j, kind in hit if kind == "key")
    return {
        "lint_only": sorted(j for j, kind in hit if kind == "code" and j not in skip),
        "reused": skip,
    }


def _live(artifacts: list[dict[str, Any]], repository_id: int) -> list[dict[str, Any]]:
    return [
        a
        for a in artifacts
        if not a["expired"] and a["workflow_run"]["head_repository_id"] == repository_id
    ]


def changes(
    ci: str,
    filters: str,
    *,
    pull_request: int,
    fetch: Callable[[str], list[dict[str, Any]]],
    repository_id: int,
    diff: Callable[[str], list[str]],
) -> dict[str, str]:
    """Returns the Go files each Go job's inputs changed since its last pass on the pull
    request, or all. diff lists the files changed since a commit."""
    jobs = load(ci)[0].get("jobs", {})
    globs = {k: paths(v) for k, v in load(filters)[0].items()}
    shared = matcher([*COMMON, WORKFLOW, FILTERS])
    out = {}
    for job, refs in reusable(jobs, globs).items():
        if jobs[job].get("uses") != GO:
            continue
        name = ARTIFACT.format(job=job, key=f"pr{pull_request}")
        passes = _live(fetch(name), repository_id) if pull_request else []
        if not passes:
            out[job] = "all"
            continue
        last = max(passes, key=lambda a: a["created_at"])["workflow_run"]["head_sha"]
        pattern = matcher(set().union(*(globs[r] for r in refs)))
        changed = [
            f
            for f in diff(last)
            if (pattern.fullmatch(f) and not f.endswith(PROSE)) or shared.fullmatch(f)
        ]
        go = all(f.endswith(".go") for f in changed)
        out[job] = " ".join(changed) if go else "all"
    return out


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


def _fetch(rev: str) -> None:
    subprocess.run(
        ["git", "fetch", "--no-tags", "--depth=1", "origin", rev], check=True
    )


def _diff(head: str) -> Callable[[str], list[str]]:
    def diff(base: str) -> list[str]:
        _fetch(base)
        return subprocess.run(
            ["git", "diff", "--name-only", "--no-renames", base, head],
            capture_output=True,
            text=True,
            check=True,
        ).stdout.splitlines()

    return diff


def main(argv: list[str]) -> None:
    ci, filters = Path(WORKFLOW).read_text(), Path(FILTERS).read_text()
    if argv[1] == "keys":
        head = os.environ.get("HEAD_SHA", "")
        if head:
            _fetch(head)
        result: Any = keys(
            ci,
            filters,
            Snapshot(tree(), lambda p: Path(p).read_bytes()),
            read=lambda p: Path(p).read_text(),
            outputs=json.loads(os.environ["FILTER_OUTPUTS"]),
            head=snapshot(head) if head else None,
        )
    elif argv[1] == "reused":
        result = reused(
            json.loads(argv[2]),
            fetch=fetch_artifacts(os.environ["GITHUB_REPOSITORY"]),
            repository_id=int(os.environ["GITHUB_REPOSITORY_ID"]),
        )
    elif argv[1] == "changes":
        if argv[3]:
            _fetch(argv[3])
        result = changes(
            ci,
            filters,
            pull_request=int(argv[2]),
            fetch=fetch_artifacts(os.environ["GITHUB_REPOSITORY"]),
            repository_id=int(os.environ["GITHUB_REPOSITORY_ID"]),
            diff=_diff(argv[3]),
        )
    else:
        raise ValueError(f"unknown command: {argv[1]}")
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    main(sys.argv)
