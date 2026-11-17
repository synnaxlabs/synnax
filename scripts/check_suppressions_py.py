#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports Python suppressions that a change adds against a base revision.

A suppression is a `# type: ignore`, `# noqa`, or `# pyright: ignore` comment, or a
`cast(...)` or `typing.cast(...)` call. Ruff and mypy only flag blanket or unused ones.
The check reads comments and calls from tokens, so a string that holds the text is not
a finding. Only added lines count.

    uv run --project scripts scripts/check_suppressions_py.py [--base REF]
"""

import argparse
import io
import re
import subprocess
import sys
import tokenize
from dataclasses import dataclass
from pathlib import Path

_COMMENT = re.compile(r"#\s*(type:\s*ignore|noqa|pyright:\s*ignore)\b")
_FILE = re.compile(r"^\+\+\+ b/(.+)$")
_HUNK = re.compile(r"^@@ -\S+ \+(\d+)(?:,(\d+))? @@")


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


def _is_cast(tokens: list[tokenize.TokenInfo], i: int) -> bool:
    if tokens[i].string != "cast" or tokens[i + 1].string != "(":
        return False
    before = tokens[i - 1].string if i > 0 else ""
    if before == "def":
        return False
    if before == ".":
        return i > 1 and tokens[i - 2].string == "typing"
    return True


def check_source(source: str, *, path: str, lines: set[int]) -> list[Finding]:
    """Returns the suppressions on the given lines of one file."""
    tokens = [
        t
        for t in tokenize.generate_tokens(io.StringIO(source).readline)
        if t.type not in (tokenize.NL, tokenize.NEWLINE, tokenize.INDENT)
    ]
    findings = []
    for i, token in enumerate(tokens[:-1]):
        line = token.start[0]
        if line not in lines:
            continue
        if token.type == tokenize.COMMENT and (m := _COMMENT.search(token.string)):
            message = f"remove the `{m.group(0)}`; fix the type or the code"
        elif token.type == tokenize.NAME and _is_cast(tokens, i):
            message = "remove the `cast`; narrow the type instead"
        else:
            continue
        findings.append(Finding(path=path, line=line, message=message))
    return findings


def check(base: str, *, root: Path) -> list[Finding]:
    """Returns the suppressions that the working tree adds against base."""
    diff = subprocess.run(
        ["git", "diff", "--unified=0", "--diff-filter=d", base, "--", "*.py"],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    ).stdout
    findings = []
    for path, lines in sorted(added_lines(diff).items()):
        source = (root / path).read_text(encoding="utf-8")
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
