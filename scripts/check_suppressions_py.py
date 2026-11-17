#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports Python suppressions that a change adds against a base revision.

A suppression is a `# type: ignore`, `# noqa`, or `# pyright: ignore` comment, a
file-level directive such as `# ruff: noqa` or `# mypy: ignore-errors`, or a call to
`typing.cast` under any name. Ruff and mypy only flag blanket or unused ones. The check
reads comments from tokens and calls from the syntax tree, so a string that holds the
text is not a finding. Only lines added to tracked files and the lines of untracked
files count. Generated files are skipped: fix their generator instead.

    uv run --project scripts scripts/check_suppressions_py.py [--base REF]
"""

import argparse
import ast
import io
import re
import subprocess
import sys
import tokenize
from dataclasses import dataclass
from pathlib import Path

_COMMENT = re.compile(
    r"#\s*(type:\s*ignore|(?:(?:ruff|flake8):\s*)?noqa|pyright:\s*ignore"
    r"|mypy:\s*(?:ignore-errors|disable-error-code)"
    r"|pyright:\s*(?:basic|report\w+\s*=\s*(?:false|none)))\b",
    re.IGNORECASE,
)
_FILE = re.compile(r"^\+\+\+ b/(.+)$")
_HUNK = re.compile(r"^@@ -\S+ \+(\d+)(?:,(\d+))? @@")
_TYPING = {"typing", "typing_extensions"}
# Git flags that fix the diff format whatever the user's configuration.
_DIFF = [
    "-c",
    "core.quotePath=false",
    "diff",
    "--no-ext-diff",
    "--no-color",
    "--src-prefix=a/",
    "--dst-prefix=b/",
    "--unified=0",
    "--diff-filter=d",
]


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    message: str


def added_lines(diff: str) -> dict[str, set[int]]:
    """Returns the added line numbers of each file in a unified diff with no context."""
    added: dict[str, set[int]] = {}
    path = ""
    for line in diff.splitlines():
        if m := _FILE.match(line):
            path = m.group(1)
        elif m := _HUNK.match(line):
            start = int(m.group(1))
            count = 1 if m.group(2) is None else int(m.group(2))
            added.setdefault(path, set()).update(range(start, start + count))
    return added


def _generated(source: str) -> bool:
    return any("Code generated" in line for line in source.splitlines()[:30])


def _cast_lines(tree: ast.Module) -> set[int]:
    names = {"cast"}
    modules = set(_TYPING)
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom) and node.module in _TYPING:
            names |= {a.asname for a in node.names if a.name == "cast" and a.asname}
        elif isinstance(node, ast.Import):
            modules |= {a.asname for a in node.names if a.name in _TYPING and a.asname}
    lines = set()
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        f = node.func
        if (isinstance(f, ast.Name) and f.id in names) or (
            isinstance(f, ast.Attribute)
            and f.attr == "cast"
            and isinstance(f.value, ast.Name)
            and f.value.id in modules
        ):
            lines.add(f.lineno)
    return lines


def check_source(source: str, *, path: str, lines: set[int]) -> list[Finding]:
    """Returns the suppressions on the given lines of one file."""
    if _generated(source):
        return []
    findings = [
        Finding(
            path=path,
            line=t.start[0],
            message=f"remove the `{m.group(0)}`; fix the type or the code",
        )
        for t in tokenize.generate_tokens(io.StringIO(source).readline)
        if t.type == tokenize.COMMENT
        and t.start[0] in lines
        and (m := _COMMENT.search(t.string))
    ]
    findings += [
        Finding(path=path, line=n, message="remove the `cast`; narrow the type instead")
        for n in _cast_lines(ast.parse(source, filename=path)) & lines
    ]
    return sorted(findings, key=lambda f: f.line)


def _git(root: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True, text=True
    ).stdout


def check(base: str, *, root: Path) -> list[Finding]:
    """Returns the suppressions that the working tree adds against base."""
    added = added_lines(_git(root, *_DIFF, base, "--", "*.py"))
    untracked = _git(root, "ls-files", "--others", "--exclude-standard", "--", "*.py")
    findings = []
    for path in sorted(added.keys() | set(untracked.splitlines())):
        source = (root / path).read_text(encoding="utf-8")
        lines = added.get(path, set(range(1, source.count("\n") + 2)))
        findings.extend(check_source(source, path=path, lines=lines))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--base", default="origin/main")
    findings = check(parser.parse_args().base, root=Path())
    for f in findings:
        print(f"{f.path}:{f.line}: {f.message}")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
