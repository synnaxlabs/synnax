#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports each test file whose count of skip markers rose since a base commit.

A skip marker skips a test or marks it pending: `Skip(` and the `P` and `X` specs of
Ginkgo, `t.Skip` in Go, `.skip(` and `xit(` in TypeScript, `pytest.mark.skip` and
`self.skipTest(` in Python, and `DISABLED_` or `GTEST_SKIP` in C++. The check compares
counts, so a marker that moves inside a file, a renamed file, or a format run is not a
finding. It reads code with comments and strings removed. A platform case uses a build
tag or a platform file, not a skip.

    uv run --project scripts scripts/check_skipped_tests.py --base <commit>
"""

import argparse
import difflib
import io
import re
import subprocess
import sys
import tokenize
from dataclasses import dataclass
from pathlib import Path

_GO = [
    re.compile(
        r"(?<![\w.])(?:ginkgo\.)?[PX](?:It|Describe|Context|When|Specify|Entry"
        r"|DescribeTable)\s*\("
    ),
    re.compile(r"(?<![\w.])(?:ginkgo\.)?Pending\b(?!\s*[:=(])"),
    re.compile(r"(?<![\w.])(?:ginkgo\.)?Skip\s*\("),
    re.compile(r"\b[tb]\.Skip(?:f|Now)?\s*\("),
]
_GO_SKIP_FUNC = re.compile(r"\bfunc\s+(?:\([^)]*\)\s*)?Skip\b")
_TS = [
    re.compile(r"\.(?:skip|skipIf|runIf|todo)\b(?=\s*[(.`])"),
    re.compile(r"(?<![\w.$])(?:xit|xdescribe|xtest)\s*\("),
]
_PY = [
    re.compile(r"\bpytest\.mark\.(?:skip|skipif|xfail)\b"),
    re.compile(r"\bpytest\.(?:skip|importorskip)\s*\("),
    re.compile(r"\bunittest\.skip\w*"),
    re.compile(r"\bself\.(?:skipTest|skip)\s*\("),
]
_CPP = [re.compile(r"\bDISABLED_\w*"), re.compile(r"\bGTEST_SKIP\b")]
_TS_TEST = re.compile(r"\.(?:spec|test)\.tsx?$|\.bench\.ts$")


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    message: str


def language(path: str) -> str | None:
    """Returns the language of a test file at path, or None for any other file."""
    name = path.rsplit("/", 1)[-1]
    if name.endswith("_test.go"):
        return "go"
    if _TS_TEST.search(name):
        return "ts"
    if name.endswith((".cpp", ".cc")) and name.rsplit(".", 1)[0].endswith("_test"):
        return "cpp"
    if name.endswith(".py") and (
        name.startswith("test_")
        or name.endswith("_test.py")
        or name == "conftest.py"
        or path.startswith("integration/tests/")
    ):
        return "py"
    return None


def _blank(text: str, start: int, end: int) -> str:
    return "".join(c if c == "\n" else " " for c in text[start:end])


def _strip_c(source: str, *, raw: str) -> str:
    """Returns source with C-style comments and string literals made blank. raw holds
    the quote characters that open a string with no escapes."""
    out, i, n = [], 0, len(source)
    while i < n:
        c, nxt = source[i], source[i + 1 : i + 2]
        if c == "/" and nxt in ("/", "*"):
            end = source.find("\n" if nxt == "/" else "*/", i + 2)
            end = n if end < 0 else end + (0 if nxt == "/" else 2)
        elif c == "R" and nxt == '"' and (i == 0 or not source[i - 1].isalnum()):
            delim_end = source.find("(", i + 2)
            close = ")" + source[i + 2 : delim_end] + '"'
            end = source.find(close, delim_end)
            end = n if end < 0 else end + len(close)
        elif c in '"`' or (c == "'" and not (i > 0 and source[i - 1].isalnum())):
            end, escapes = i + 1, c not in raw
            while end < n and source[end] != c:
                end += 2 if escapes and source[end] == "\\" else 1
            end += 1
        else:
            out.append(c)
            i += 1
            continue
        out.append(_blank(source, i, min(end, n)))
        i = end
    return "".join(out)


def _strip_py(source: str) -> str:
    """Returns source with Python comments and string literals made blank."""
    lines = source.splitlines(keepends=True)
    try:
        tokens = list(tokenize.generate_tokens(io.StringIO(source).readline))
    except (tokenize.TokenError, SyntaxError):
        return source
    blanked = {tokenize.COMMENT, tokenize.STRING}
    blanked |= {
        getattr(tokenize, kind)
        for part in ("START", "MIDDLE", "END")
        for kind in (f"FSTRING_{part}", f"TSTRING_{part}")
        if hasattr(tokenize, kind)
    }
    starts = [0]
    for line in lines:
        starts.append(starts[-1] + len(line))
    chars = list(source)
    for t in tokens:
        if t.type in blanked:
            start = starts[t.start[0] - 1] + t.start[1]
            end = starts[t.end[0] - 1] + t.end[1]
            chars[start:end] = _blank(source, start, end)
    return "".join(chars)


def markers(source: str, lang: str) -> dict[int, list[str]]:
    """Returns the skip markers in source by line, after comments and strings."""
    if lang == "py":
        code, patterns = _strip_py(source), _PY
    elif lang == "go":
        code = _GO_SKIP_FUNC.sub(
            lambda m: _blank(m.group(0), 0, len(m.group(0))), _strip_c(source, raw="`")
        )
        patterns = _GO
    else:
        code = _strip_c(source, raw="")
        patterns = _TS if lang == "ts" else _CPP
    found: dict[int, list[str]] = {}
    for number, line in enumerate(code.splitlines(), start=1):
        hits = [m.group(0).strip() for p in patterns for m in p.finditer(line)]
        if hits:
            found[number] = hits
    return found


def _git(root: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=root, check=True, capture_output=True
    ).stdout.decode("utf-8", errors="replace")


def _changed(root: Path, base: str) -> list[tuple[str | None, str, str]]:
    """Returns each test file that HEAD adds or changes since base: its path at base, or
    None when base has no such file, its path at HEAD, and its language."""
    fields = _git(root, "diff", "--name-status", "-z", "-M", base, "HEAD").split("\0")
    changed, i = [], 0
    while i < len(fields) and fields[i]:
        status = fields[i]
        if status[0] in "RC":
            old, new, i = fields[i + 1], fields[i + 2], i + 3
        else:
            old, new, i = fields[i + 1], fields[i + 1], i + 2
        lang = language(new)
        if status[0] == "D" or lang is None:
            continue
        changed.append((None if status[0] in "AC" else old, new, lang))
    return changed


def check(*, root: Path, base: str) -> list[Finding]:
    """Returns a finding for each test file whose count of skip markers rose."""
    findings = []
    for old, new, lang in _changed(root, base):
        before = _git(root, "show", f"{base}:{old}") if old is not None else ""
        after = _git(root, "show", f"HEAD:{new}")
        was, now = markers(before, lang), markers(after, lang)
        count_was, count_now = sum(map(len, was.values())), sum(map(len, now.values()))
        if count_now <= count_was:
            continue
        note = f"skip markers in the file: {count_was} before, {count_now} now"
        matcher = difflib.SequenceMatcher(
            None, before.splitlines(), after.splitlines(), autojunk=False
        )
        added = [
            line
            for tag, _, _, j1, j2 in matcher.get_opcodes()
            if tag in ("insert", "replace")
            for line in range(j1 + 1, j2 + 1)
            if line in now
        ]
        findings += [
            Finding(
                path=new,
                line=line,
                message=f"adds a skipped test: `{', '.join(now[line])}` ({note})",
            )
            for line in added
        ] or [Finding(path=new, line=0, message=f"adds a skipped test ({note})")]
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument(
        "--base",
        help="the commit to compare with; the default is the merge base with "
        "origin/main",
    )
    args = parser.parse_args()
    root = Path()
    base = args.base or _git(root, "merge-base", "origin/main", "HEAD").strip()
    findings = check(root=root, base=base)
    for f in findings:
        print(f"{f.path}:{f.line}: {f.message}")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
