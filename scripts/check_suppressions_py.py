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
`typing.cast` under any name. Any use of `TYPE_CHECKING` counts too, since it hides
imports from the runtime. Ruff and mypy only flag blanket or unused suppressions. The
check reads comments and names from tokens and calls from the syntax tree, so a string
that holds the text is not a finding. Only lines added to tracked files and the lines of
untracked files count, and only for a kind whose count in the file the change raises, so
an edit to a line that already holds one passes. Generated files are skipped: fix their
generator instead.

    uv run --project scripts scripts/check_suppressions_py.py [--base REF]
"""

import argparse
import ast
import io
import re
import subprocess
import sys
import tokenize
from collections import Counter
from dataclasses import dataclass
from pathlib import Path

_COMMENT = re.compile(
    r"#\s*(type:\s*ignore|(?:(?:ruff|flake8):\s*)?noqa|pyright:\s*ignore"
    r"|mypy:\s*(?:ignore-errors|disable-error-code)"
    r"|pyright:\s*(?:basic|report\w+\s*=\s*(?:false|none)))\b",
    re.IGNORECASE,
)
_OLD = re.compile(r"^--- a/(.+)$")
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
    "--find-renames",
    "--diff-filter=d",
]


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    kind: str
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


def old_paths(diff: str) -> dict[str, str]:
    """Returns the path at the base of each file in a unified diff, by its new path."""
    old: dict[str, str] = {}
    source = ""
    for line in diff.splitlines():
        if m := _OLD.match(line):
            source = m.group(1)
        elif m := _FILE.match(line):
            if source:
                old[m.group(1)] = source
            source = ""
    return old


def _generated(source: str) -> bool:
    return any(line.startswith("# Code generated") for line in source.splitlines()[:30])


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


def _findings(source: str, *, path: str, lines: set[int] | None) -> list[Finding]:
    """Returns the suppressions on the given lines, or on every line for None."""
    tokens = [
        t
        for t in tokenize.generate_tokens(io.StringIO(source).readline)
        if lines is None or t.start[0] in lines
    ]
    findings = [
        Finding(
            path=path,
            line=t.start[0],
            kind="comment",
            message=f"remove the `{m.group(0)}`; fix the type or the code",
        )
        for t in tokens
        if t.type == tokenize.COMMENT and (m := _COMMENT.search(t.string))
    ]
    findings += [
        Finding(
            path=path,
            line=t.start[0],
            kind="TYPE_CHECKING",
            message="remove the `TYPE_CHECKING` guard; import at runtime and break "
            "any import cycle instead",
        )
        for t in tokens
        if t.type == tokenize.NAME and t.string == "TYPE_CHECKING"
    ]
    casts = _cast_lines(ast.parse(source, filename=path))
    findings += [
        Finding(
            path=path,
            line=n,
            kind="cast",
            message="remove the `cast`; narrow the type instead",
        )
        for n in (casts if lines is None else casts & lines)
    ]
    return sorted(findings, key=lambda f: f.line)


def check_source(
    source: str, *, path: str, lines: set[int], base: str = ""
) -> list[Finding]:
    """Returns the suppressions on the given lines of one file.

    A kind of suppression is reported only when the file has more of it than base, so
    an edit to a line that already holds one passes.
    """
    if _generated(source):
        return []
    total = Counter(f.kind for f in _findings(source, path=path, lines=None))
    before = Counter(f.kind for f in _findings(base, path=path, lines=None))
    return [
        f
        for f in _findings(source, path=path, lines=lines)
        if total[f.kind] > before[f.kind]
    ]


def _git(root: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True, text=True
    ).stdout


def _base(root: Path, base: str, path: str) -> str:
    """Returns the file at base, or nothing when base lacks it."""
    shown = subprocess.run(
        ["git", "show", f"{base}:{path}"], cwd=root, capture_output=True, text=True
    )
    return shown.stdout if shown.returncode == 0 else ""


def check(base: str, *, root: Path) -> list[Finding]:
    """Returns the suppressions that the working tree adds against base."""
    diff = _git(root, *_DIFF, base, "--", "*.py")
    added = added_lines(diff)
    old = old_paths(diff)
    untracked = _git(root, "ls-files", "--others", "--exclude-standard", "--", "*.py")
    findings = []
    for path in sorted(added.keys() | set(untracked.splitlines())):
        source = (root / path).read_text(encoding="utf-8")
        if path in added:
            lines = added[path]
            base_source = _base(root, base, old.get(path, path))
        else:
            lines = set(range(1, source.count("\n") + 2))
            base_source = ""
        findings.extend(check_source(source, path=path, lines=lines, base=base_source))
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
