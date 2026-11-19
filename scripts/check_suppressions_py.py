#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports the Python suppressions in every tracked and untracked file.

A suppression is a `# type: ignore`, `# noqa`, or `# pyright: ignore` comment, a
file-level directive such as `# ruff: noqa` or `# mypy: ignore-errors`, a call to
`typing.cast` under any name, or any use of `TYPE_CHECKING`. The check reads comments
and names from tokens and calls from the syntax tree, so a string that holds the text is
not a finding. Generated files are skipped: fix their generator instead.

    uv run --project scripts scripts/check_suppressions_py.py
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
_TYPING = {"typing", "typing_extensions"}


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    message: str


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


def check_source(source: str, *, path: str) -> list[Finding]:
    """Returns the suppressions in one file."""
    if _generated(source):
        return []
    tokens = list(tokenize.generate_tokens(io.StringIO(source).readline))
    findings = [
        Finding(
            path=path,
            line=t.start[0],
            message=f"remove the `{m.group(0)}`; fix the type or the code",
        )
        for t in tokens
        if t.type == tokenize.COMMENT and (m := _COMMENT.search(t.string))
    ]
    findings += [
        Finding(
            path=path,
            line=t.start[0],
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
            message="remove the `cast`; narrow the type instead",
        )
        for n in casts
    ]
    return sorted(findings, key=lambda f: f.line)


def check(*, root: Path) -> list[Finding]:
    """Returns the suppressions in the tracked and untracked files under root."""
    paths = subprocess.run(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "*.py"],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.split("\0")
    findings = []
    for path in sorted(set(paths) - {""}):
        file = root / path
        if file.is_file():
            source = file.read_text(encoding="utf-8")
            findings.extend(check_source(source, path=path))
    return findings


def main() -> int:
    argparse.ArgumentParser(description=__doc__.splitlines()[0]).parse_args()
    findings = check(root=Path())
    for f in findings:
        print(f"{f.path}:{f.line}: {f.message}")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
