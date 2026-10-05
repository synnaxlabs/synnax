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
    [--manual-pass <skip reason>]... [--manual-note <why they were run by hand>]

The data goes to DigitalOcean Spaces, where the docs build reads it; redeploy the docs
to publish it. --out writes it to a local file instead. Uploading needs DO_SPACES_KEY and
DO_SPACES_SECRET in the environment.

Integration results come from the run's `test-results-*` artifacts. Unit suites write no
per-test report yet, so their tests are enumerated from source at the run's commit and
take their job's conclusion.
"""

import argparse
import ast
import json
import os
import re
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any

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

# language -> the framework its unit tests run on.
FRAMEWORKS = {"go": "ginkgo", "ts": "vitest", "cpp": "gtest", "py": "pytest"}


# Runner OS -> what platform.system().lower() returns on it.
PLATFORMS = {"ubuntu": "linux", "windows": "windows", "macos": "darwin"}

TC_STATES = {
    "PASSED": "passed",
    "FLAKY": "flaky",
    "FAILED": "failed",
    "TIMEOUT": "failed",
    "KILLED": "failed",
}

# Matrix jobs append their runner, as in "core / Test (ubuntu-latest)"; single jobs do
# not. The runner's OS comes from the job's labels either way.
JOB_NAME = re.compile(r"^([a-z0-9-]+) / Test(?: \(.+\))?$")
# An integration job that also runs other products' tests lists them, as in
# "integration / Test (windows) / arc + console".
TC_JOB_NAME = re.compile(r"^integration / Test \((\w+)\) / (\w+)(?: \+ [\w +]+)?$")


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


def integration_products(sources: dict[str, str]) -> dict[str, str]:
    """Returns PRODUCTS from integration/jobs.py in sources: the product each test
    file covers, by the file's prefix."""
    for node in ast.parse(sources["jobs.py"]).body:
        if isinstance(node, ast.Assign) and any(
            isinstance(t, ast.Name) and t.id == "PRODUCTS" for t in node.targets
        ):
            products: dict[str, str] = ast.literal_eval(node.value)
            return products
    raise ValueError("integration/jobs.py has no PRODUCTS")


def integration_sources(root: Path, sha: str) -> dict[str, str]:
    """Maps each Python file under integration/ at sha, by its path there, to its
    source."""
    paths = [
        p
        for p in subprocess.run(
            ["git", "ls-tree", "-r", "--name-only", sha, "--", "integration"],
            cwd=root,
            check=True,
            capture_output=True,
            text=True,
        ).stdout.splitlines()
        if p.endswith(".py")
    ]
    out = subprocess.run(
        ["git", "cat-file", "--batch"],
        cwd=root,
        input="".join(f"{sha}:{p}\n" for p in paths).encode(),
        check=True,
        capture_output=True,
    ).stdout
    sources = {}
    for path in paths:
        header, out = out.split(b"\n", 1)
        size = int(header.split()[2])
        sources[path.removeprefix("integration/")] = out[:size].decode()
        out = out[size + 1 :]
    return sources


@dataclass
class Case:
    """An integration case's class, as read from source."""

    source: str
    tags: list[str]
    # Runner OS -> why the case auto-passes there instead of running.
    skipped: dict[str, str]


class Cases:
    """Resolves integration cases to their classes and tags them by ancestry."""

    def __init__(self, sources: dict[str, str]) -> None:
        self.trees = {path: ast.parse(src) for path, src in sources.items()}
        self.configures = {
            p for p, src in sources.items() if ".tasks.configure(" in src
        }

    def _module(self, name: str) -> str | None:
        base = name.replace(".", "/")
        for path in (f"{base}.py", f"{base}/__init__.py"):
            if path in self.trees:
                return path
        return None

    def _find(self, path: str, name: str) -> tuple[str, ast.ClassDef] | None:
        """Finds the class a name refers to in a module, following imports. None
        means a class from outside integration/, such as ABC."""
        body = self.trees[path].body
        for node in body:
            if isinstance(node, ast.ClassDef) and node.name == name:
                return path, node
        for node in body:
            if isinstance(node, ast.ImportFrom) and node.module is not None:
                for alias in node.names:
                    if (alias.asname or alias.name) == name:
                        module = self._module(node.module)
                        return (
                            None if module is None else self._find(module, alias.name)
                        )
        return None

    def _ancestry(self, path: str, cls: ast.ClassDef) -> list[tuple[str, ast.ClassDef]]:
        chain = [(path, cls)]
        for base in cls.bases:
            if isinstance(base, ast.Name):
                hit = self._find(path, base.id)
                if hit is not None:
                    chain += self._ancestry(*hit)
        return chain

    def _class(self, case: str, name: str) -> tuple[str, ast.ClassDef]:
        path = f"tests/{case}.py"
        classes = [n for n in self.trees[path].body if isinstance(n, ast.ClassDef)]
        for cls in classes:
            if cls.name == name:
                return path, cls
        # A module holding one test class names its case after the module or the
        # sequence entry, not the class.
        tests = [
            c
            for c in classes
            if not c.name.startswith("_")
            and "TestCase" in {a.name for _, a in self._ancestry(path, c)}
        ]
        if len(tests) != 1:
            raise ValueError(f"no class for {case} › {name}")
        return path, tests[0]

    def resolve(self, case: str, name: str) -> Case:
        path, cls = self._class(case, name)
        chain = self._ancestry(path, cls)
        names = {c.name for _, c in chain}
        skipped = {}
        for _, c in chain:
            for node in ast.walk(c):
                if isinstance(node, ast.If):
                    for os_name, system in PLATFORMS.items():
                        reason = auto_pass(node, system)
                        if reason is not None:
                            skipped.setdefault(os_name, reason)
        tags = [
            "playwright" if "ConsoleCase" in names else "headless",
            # Simulated devices connect through the Driver, and a configured task
            # runs on it.
            "driver"
            if "SimulatorCase" in names or any(p in self.configures for p, _ in chain)
            else "no-driver",
        ]
        runs = set(PLATFORMS) - set(skipped)
        if runs == {"windows"}:
            tags.append("windows-only")
        elif "windows" not in runs:
            tags.append("not-windows")
        return Case(f"integration/{path}#L{cls.lineno}", tags, skipped)


def auto_pass(node: ast.If, system: str) -> str | None:
    """Returns the auto_pass message when node gates an auto_pass on the platform
    and the gate holds for system, as in
    `if platform.system().lower() != "windows": self.auto_pass(msg=...)`."""
    test = node.test
    if not (
        isinstance(test, ast.Compare)
        and len(test.ops) == 1
        and isinstance(test.ops[0], (ast.Eq, ast.NotEq))
        and isinstance(test.comparators[0], ast.Constant)
        and "system" in ast.unparse(test.left)
    ):
        return None
    if (system == test.comparators[0].value) != isinstance(test.ops[0], ast.Eq):
        return None
    for stmt in node.body:
        for call in ast.walk(stmt):
            if (
                isinstance(call, ast.Call)
                and isinstance(call.func, ast.Attribute)
                and call.func.attr == "auto_pass"
            ):
                msg = call.keywords[0].value if call.keywords else call.args[0]
                return msg.value if isinstance(msg, ast.Constant) else "Auto-passed"
    return None


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
    parser.add_argument(
        "--manual-pass",
        action="append",
        default=[],
        metavar="REASON",
        help="skip reason whose tests someone ran by hand, and all passed",
    )
    parser.add_argument(
        "--manual-note",
        help="why the --manual-pass tests were run by hand, shown on the page",
    )
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
            ".jobs[]|{name,conclusion,html_url,labels}|tojson",
        ).splitlines()
    ]

    suites: list[dict] = []
    tests: list[list] = []

    for job in jobs:
        m = JOB_NAME.match(job["name"])
        if m is None or m.group(1) not in JOBS:
            continue
        key = m.group(1)
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
                "tags": ["unit", FRAMEWORKS[lang]],
                "os": os_of(job["labels"][0]),
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

    sources = integration_sources(ROOT, sha)
    cases = Cases(sources)
    products = integration_products(sources)
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
            job = tc_jobs[(runner_os, target)]
            # Each test conductor run of the job writes its own summary, and a job can
            # run the tests of several products.
            by_product: dict[str, list[dict[str, Any]]] = {}
            for path in sorted(art.glob("run-*/summary.json")):
                for t in json.loads(path.read_text())["tests"]:
                    by_product.setdefault(products[t["file"]], []).append(t)
            for product, results in sorted(by_product.items()):
                suites.append(
                    {
                        "job": f"integration-{target}",
                        "product": product,
                        "lang": "py",
                        "kind": "system",
                        "tags": ["e2e"],
                        "os": runner_os,
                        "conclusion": job["conclusion"],
                        "url": job["html_url"],
                        "count": len(results),
                    }
                )
                s = len(suites) - 1
                for t in results:
                    case = cases.resolve(t["case"], t["name"])
                    state = TC_STATES[t["status"]]
                    message = t.get("error_message")
                    # tc reports an auto-passed case as PASSED.
                    if runner_os in case.skipped:
                        state, message = "skipped", case.skipped[runner_os]
                    tests.append(
                        [
                            s,
                            f"{t['case']} › {t['name']}",
                            state,
                            message,
                            t.get("duration_s"),
                            case.source,
                            case.tags,
                        ]
                    )

    if bool(args.manual_pass) != (args.manual_note is not None):
        sys.exit("--manual-pass and --manual-note go together")
    reasons = {t[3] for t in tests if t[2] == "skipped"}
    for reason in args.manual_pass:
        if reason not in reasons:
            sys.exit(f"no skipped tests with reason {reason!r}: {sorted(reasons)}")

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
            "manual_passes": args.manual_pass,
            "manual_note": args.manual_note,
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
