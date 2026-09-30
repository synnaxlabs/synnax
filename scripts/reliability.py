#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# /// script
# dependencies = ["boto3"]
# ///

"""Builds the reliability page data from a CI workflow run and uploads it.

Usage: uv run scripts/reliability.py <run-url> --version 0.59.0 [--out path]

The data goes to DigitalOcean Spaces, where the docs build reads it; redeploy the docs
to publish it. --out writes it to a local file instead. Uploading needs DO_SPACES_KEY and
DO_SPACES_SECRET in the environment.

Integration results come from the run's `test-results-*` artifacts. Unit suites write no
per-test report yet, so their tests are enumerated from source at the run's commit and
take their job's conclusion.
"""

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

REPO = "synnaxlabs/synnax"
ROOT = Path(__file__).resolve().parent.parent
SPACES_ENDPOINT = "https://nyc3.digitaloceanspaces.com"
SPACES_BUCKET = "synnax"
SPACES_KEY = "docs/reliability/run.json"

PRODUCTS = [
    ("core", "Core"),
    ("console", "Console"),
    ("driver", "Driver"),
    ("arc", "Arc"),
    ("clients", "Client libraries"),
    ("toolchain", "Toolchain"),
]

# CI job key -> (product, language, source directories).
JOBS: dict[str, tuple[str, str, list[str]]] = {
    "core": ("core", "go", ["core"]),
    "cesium": ("core", "go", ["cesium"]),
    "aspen": ("core", "go", ["aspen"]),
    "x-go": ("core", "go", ["x/go"]),
    "freighter-go": ("core", "go", ["freighter/go"]),
    "freighter-integration": ("core", "go", ["freighter/integration"]),
    "alamos-go": ("core", "go", ["alamos/go"]),
    "arc-go": ("arc", "go", ["arc/go"]),
    "arc-cpp": ("arc", "cpp", ["arc/cpp"]),
    "driver": ("driver", "cpp", ["driver"]),
    "x-cpp": ("driver", "cpp", ["x/cpp"]),
    "freighter-cpp": ("driver", "cpp", ["freighter/cpp"]),
    "soem": ("driver", "cpp", ["vendor/soem"]),
    "console": ("console", "ts", ["console"]),
    "pluto": ("console", "ts", ["pluto"]),
    "lyra": ("console", "ts", ["lyra"]),
    "drift": ("console", "ts", ["drift"]),
    "x-ts": ("console", "ts", ["x/ts"]),
    "alamos-ts": ("console", "ts", ["alamos/ts"]),
    "client-ts": ("clients", "ts", ["client/ts"]),
    "client-py": ("clients", "py", ["client/py"]),
    "client-cpp": ("clients", "cpp", ["client/cpp", "client/clib"]),
    "freighter-ts": ("clients", "ts", ["freighter/ts"]),
    "freighter-py": ("clients", "py", ["freighter/py"]),
    "x-py": ("clients", "py", ["x/py"]),
    "alamos-py": ("clients", "py", ["alamos/py"]),
    "oracle": ("toolchain", "go", ["oracle"]),
    "docs": ("toolchain", "ts", ["site/docs"]),
    "vite-plugin": ("toolchain", "ts", ["configs/vite"]),
    "scripts": ("toolchain", "py", ["scripts", ".github/scripts"]),
}

# Integration matrix target -> product.
TARGETS = {"console": "console", "driver": "driver", "arc": "arc"}

# language -> (pathspec globs, declaration pattern). The first group is the name.
LANGS = {
    "go": (["*_test.go"], r"^\s*(?:It|Entry)\(\s*(?:\"((?:[^\"\\]|\\.)*)\")?"),
    "ts": (
        ["*.spec.ts", "*.spec.tsx", "*.test.ts", "*.test.tsx"],
        r"^\s*(?:it|test)(?:\.each\(.*?\))?\(\s*(?:[\"'`](.*?)[\"'`])?",
    ),
    "cpp": (["*test*.cpp"], r"^\s*TEST(?:_F|_P)?\(\s*(\w+\s*,\s*\w+)\s*\)"),
    "py": (["test_*.py", "*_test.py"], r"^\s*(?:async\s+)?def\s+(test_\w+)"),
}

FIRST_STRING = re.compile(r"[\"'`]((?:[^\"'`\\]|\\.)+)[\"'`]")

TC_STATES = {
    "PASSED": "passed",
    "FLAKY": "flaky",
    "FAILED": "failed",
    "TIMEOUT": "failed",
    "KILLED": "failed",
}

JOB_NAME = re.compile(r"^([a-z0-9-]+) / Test \(([^)]+)\)$")
TC_JOB_NAME = re.compile(r"^integration / Test \((\w+)\) / (\w+)$")


def gh(*args: str) -> str:
    return subprocess.run(
        ["gh", *args], check=True, capture_output=True, text=True
    ).stdout


def os_of(runner: str) -> str:
    for name in ("ubuntu", "windows", "macos"):
        if runner.startswith(name):
            return name
    raise ValueError(f"unknown runner: {runner}")


def enumerate_tests(
    root: Path, sha: str, lang: str, dirs: list[str]
) -> list[tuple[str, int, str]]:
    """Lists (path, line, name) for each test declared under dirs at sha."""
    globs, pattern = LANGS[lang]
    specs = [f"{d}/**/{g}" for d in dirs for g in globs]
    out = subprocess.run(
        [
            "git",
            "grep",
            "-n",
            "-P",
            "--full-name",
            pattern,
            sha,
            "--",
            *[f":(glob){s}" for s in specs],
        ],
        cwd=root,
        capture_output=True,
        text=True,
        check=False,
    )
    # git grep exits 1 when nothing matches.
    if out.returncode not in (0, 1):
        raise RuntimeError(out.stderr)
    regex = re.compile(pattern)
    files: dict[str, list[str]] = {}
    tests = []
    for line in out.stdout.splitlines():
        _, path, lineno, text = line.split(":", 3)
        m = regex.search(text)
        if m is None:
            continue
        name = m.group(1)
        if name is None:
            # The name sits on a following line, as in a wrapped It( or Entry( call.
            if path not in files:
                files[path] = subprocess.run(
                    ["git", "show", f"{sha}:{path}"],
                    cwd=root,
                    check=True,
                    capture_output=True,
                    text=True,
                ).stdout.splitlines()
            after = "\n".join(files[path][int(lineno) : int(lineno) + 3])
            literal = FIRST_STRING.search(after)
            name = literal.group(1) if literal else f"line {lineno}"
        tests.append((path, int(lineno), name))
    return tests


def tc_source(root: Path, sha: str, test: dict) -> str:
    """Links an integration case to its file, at the class named for the test."""
    path = f"integration/tests/{test['case']}.py"
    out = subprocess.run(
        ["git", "grep", "-n", "-P", rf"^class {test['name']}\b", sha, "--", path],
        cwd=root,
        capture_output=True,
        text=True,
        check=False,
    )
    if out.returncode not in (0, 1):
        raise RuntimeError(out.stderr)
    lines = out.stdout.splitlines()
    return f"{path}#L{lines[0].split(':')[2]}" if lines else path


def upload(body: str) -> None:
    key = os.environ.get("DO_SPACES_KEY")
    secret = os.environ.get("DO_SPACES_SECRET")
    if key is None or secret is None:
        sys.exit("DO_SPACES_KEY and DO_SPACES_SECRET must be set")
    # Imported here so the tests, which never upload, run without boto3.
    import boto3

    boto3.client(
        "s3",
        endpoint_url=SPACES_ENDPOINT,
        region_name="us-east-1",
        aws_access_key_id=key,
        aws_secret_access_key=secret,
    ).put_object(
        Bucket=SPACES_BUCKET,
        Key=SPACES_KEY,
        Body=body.encode(),
        ACL="public-read",
        ContentType="application/json",
        # The docs build must read the newest upload, never a cached copy.
        CacheControl="no-cache",
    )


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("run_url")
    parser.add_argument("--version", required=True, help="release the run tested")
    parser.add_argument("--out", type=Path, help="write here instead of uploading")
    args = parser.parse_args()
    run_id = re.search(r"/runs/(\d+)", args.run_url)
    if run_id is None:
        sys.exit(f"not a workflow run URL: {args.run_url}")
    rid = run_id.group(1)

    run = json.loads(gh("api", f"repos/{REPO}/actions/runs/{rid}"))
    sha = run["head_sha"]
    subprocess.run(["git", "fetch", "-q", "origin", sha], cwd=ROOT, check=True)
    jobs = [
        json.loads(line)
        for line in gh(
            "api",
            "--paginate",
            f"repos/{REPO}/actions/runs/{rid}/jobs?per_page=100",
            "-q",
            ".jobs[]|{name,conclusion,html_url}|tojson",
        ).splitlines()
    ]

    suites: list[dict] = []
    tests: list[list] = []

    for job in jobs:
        m = JOB_NAME.match(job["name"])
        if m is None or m.group(1) not in JOBS:
            continue
        key, runner = m.groups()
        product, lang, dirs = JOBS[key]
        conclusion = job["conclusion"]
        if conclusion == "skipped":
            continue
        state = {"success": "passed", "failure": "failed"}.get(conclusion, "unknown")
        found = enumerate_tests(ROOT, sha, lang, dirs)
        suites.append(
            {
                "job": key,
                "product": product,
                "lang": lang,
                "kind": "unit",
                "os": os_of(runner),
                "conclusion": conclusion,
                "url": job["html_url"],
                "count": len(found),
            }
        )
        s = len(suites) - 1
        for path, line, name in found:
            # A failed job's log does not say which of its tests failed.
            tests.append(
                [
                    s,
                    f"{path} › {name}",
                    "unknown" if state == "failed" else state,
                    None,
                    None,
                    f"{path}#L{line}",
                ]
            )

    tc_jobs = {}
    for job in jobs:
        m = TC_JOB_NAME.match(job["name"])
        if m:
            tc_jobs[(m.group(1), m.group(2))] = job

    with tempfile.TemporaryDirectory() as tmp:
        subprocess.run(
            [
                "gh",
                "run",
                "download",
                rid,
                "-R",
                REPO,
                "-p",
                "test-results-*",
                "-D",
                tmp,
            ],
            check=True,
        )
        for art in sorted(Path(tmp).iterdir()):
            # test-results-<runner>-<target>, e.g. test-results-ubuntu-build-bot-arc
            target = art.name.rsplit("-", 1)[1]
            runner_os = os_of(art.name.removeprefix("test-results-"))
            summaries = sorted(art.glob("run-*/summary.json"))
            if not summaries:
                continue
            summary = json.loads(summaries[-1].read_text())
            job = tc_jobs[(runner_os, target)]
            suites.append(
                {
                    "job": f"integration-{target}",
                    "product": TARGETS[target],
                    "lang": "py",
                    "kind": "system",
                    "os": runner_os,
                    "conclusion": job["conclusion"],
                    "url": job["html_url"],
                    "count": len(summary["tests"]),
                }
            )
            s = len(suites) - 1
            for t in summary["tests"]:
                state = TC_STATES[t["status"]]
                message = t.get("error_message")
                # tc reports auto-passed NI cases as PASSED; they need Windows DAQmx.
                if runner_os != "windows" and re.search(r"/(ni_|driver_ni)", t["case"]):
                    state, message = "skipped", "Requires Windows NI-DAQmx drivers"
                tests.append(
                    [
                        s,
                        f"{t['case']} › {t['name']}",
                        state,
                        message,
                        t.get("duration_s"),
                        tc_source(ROOT, sha, t),
                    ]
                )

    data = {
        "version": 1,
        "run": {
            "id": int(rid),
            "url": run["html_url"],
            "sha": sha,
            "version": args.version,
            "branch": run["head_branch"],
            "event": run["event"],
            "started_at": run["created_at"],
            "ended_at": run["updated_at"],
            "conclusion": run["conclusion"],
            "jobs": len(jobs),
        },
        "products": [{"key": k, "name": n} for k, n in PRODUCTS],
        "suites": suites,
        "tests": tests,
    }
    body = json.dumps(data, separators=(",", ":"))
    if args.out is not None:
        args.out.write_text(body)
        dest = str(args.out)
    else:
        upload(body)
        dest = f"s3://{SPACES_BUCKET}/{SPACES_KEY}"
    counts: dict[str, int] = {}
    for t in tests:
        counts[t[2]] = counts.get(t[2], 0) + 1
    print(f"{len(tests)} tests from {len(suites)} suites -> {dest}: {counts}")


if __name__ == "__main__":
    main()
