#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports a change that adds skip markers to the test files that it touches.

A skip marker skips a test or marks it pending: `Skip(` and the `P` and `X` specs of
Ginkgo, `t.Skip` in Go, `.skip(` and `xit(` in TypeScript, `pytest.mark.skip` and
`self.skipTest(` in Python, and a `DISABLED_` test name or `GTEST_SKIP` in C++. The
check compares the sum of the markers in the test files that the change touches with
the sum at the base, so a marker that moves inside a file or to another file, a renamed
file, or a format run is not a finding. It reads code with comments and strings removed.
A platform case uses a build tag, a platform file, or a platform file suffix, never a
skip.

    uv run --project scripts scripts/check_skipped_tests.py --base <commit>
"""

import argparse
import difflib
import io
import re
import subprocess
import sys
import tokenize
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

_GO = (
    re.compile(
        r"(?<![\w.])(?:ginkgo\.)?[PX](?:It|Describe|Context|When|Specify|Entry"
        r"|DescribeTable|DescribeTableSubtree)\s*\("
    ),
    re.compile(r"(?<![\w.])(?:ginkgo\.)?Pending\b(?!\s*[:=(])"),
    re.compile(r"(?<![\w.])(?:ginkgo\.)?Skip\s*\("),
    re.compile(r"(?:\b(?:t|b|tb)|\bGinkgoT\(\)|\.T\(\))\.Skip(?:f|Now)?\s*\("),
)
_GO_SKIP_FUNC = re.compile(r"\bfunc\s+(?:\([^)]*\)\s*)?Skip\b")
_TS = (
    re.compile(r"\.(?:skip|skipIf|runIf|todo)\b(?=\s*[(.`])"),
    re.compile(r"(?<![\w.$])(?:xit|xdescribe|xtest)\s*\("),
)
_PY = (
    re.compile(r"\bpytest\.mark\.(?:skip|skipif|xfail)\b"),
    re.compile(r"\bpytest\.(?:skip|importorskip|xfail)\s*\("),
    re.compile(r"\bunittest\.(?:skip\w*|SkipTest|expectedFailure)\b"),
    re.compile(r"\bself\.skipTest\s*\("),
)
_INTEGRATION = (*_PY, re.compile(r"\bself\.skip\s*\("))
_GTEST = r"\b(?:TEST|TEST_F|TEST_P|TYPED_TEST|TYPED_TEST_P)\s*\(\s*"
_CPP = (
    re.compile(_GTEST + r"(?:DISABLED_\w*|\w+\s*,\s*DISABLED_\w*)"),
    re.compile(r"\bGTEST_SKIP\b"),
)
_TS_TEST = re.compile(r"\.(?:spec|test)\.tsx?$|\.bench\.tsx?$")
_REGEX_AFTER = set("([{,;:=!&|?+-*%<>~^}")
_REGEX_KEYWORDS = {
    "return", "typeof", "case", "in", "of", "delete", "void", "throw", "new", "else",
    "do", "yield", "await",
}  # fmt: skip
_CPP_RAW = {"R", "u8R", "uR", "UR", "LR"}


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    message: str


def language(path: str) -> str | None:
    """Returns the language of a test file at path, or None for any other file. A
    Python file under integration/tests is "integration": its TestCase has `self.skip(`.
    """
    name = path.rsplit("/", 1)[-1]
    if name.endswith("_test.go"):
        return "go"
    if _TS_TEST.search(name):
        return "ts"
    if name.endswith((".cpp", ".cc")) and name.rsplit(".", 1)[0].endswith("_test"):
        return "cpp"
    if not name.endswith(".py"):
        return None
    if path.startswith("integration/tests/"):
        return "integration"
    if name.startswith("test_") or name.endswith("_test.py") or name == "conftest.py":
        return "py"
    return None


class _Code:
    """Source text whose comments and literals a lexer makes blank."""

    def __init__(self, source: str) -> None:
        self.source = source
        self.chars = list(source)

    def blank(self, start: int, end: int) -> None:
        for i in range(start, min(end, len(self.chars))):
            if self.chars[i] != "\n":
                self.chars[i] = " "

    def text(self) -> str:
        return "".join(self.chars)

    def comment(self, i: int) -> int:
        """Makes the comment at i blank and returns the index after it, or returns i
        when no comment starts there."""
        s = self.source
        if s.startswith("//", i):
            end = s.find("\n", i)
            end = len(s) if end < 0 else end
        elif s.startswith("/*", i):
            end = s.find("*/", i + 2)
            end = len(s) if end < 0 else end + 2
        else:
            return i
        self.blank(i, end)
        return end

    def quoted(self, i: int, *, escapes: bool) -> int:
        """Makes the literal that the quote at i opens blank, and returns the index
        after its closing quote."""
        s, quote, end = self.source, self.source[i], i + 1
        while end < len(s) and s[end] != quote:
            end += 2 if escapes and s[end] == "\\" else 1
        end = min(end + 1, len(s))
        self.blank(i, end)
        return end


def _token_before(source: str, i: int) -> str:
    """Returns the identifier or number that ends at i."""
    start = i
    while start > 0 and (source[start - 1].isalnum() or source[start - 1] in "_'."):
        start -= 1
    return source[start:i]


def _strip_go(source: str) -> str:
    code, i = _Code(source), 0
    while i < len(source):
        if (end := code.comment(i)) != i:
            i = end
        elif source[i] in "\"'`":
            i = code.quoted(i, escapes=source[i] != "`")
        else:
            i += 1
    return _GO_SKIP_FUNC.sub(lambda m: " " * len(m.group(0)), code.text())


def _strip_cpp(source: str) -> str:
    code, i = _Code(source), 0
    while i < len(source):
        c = source[i]
        if (end := code.comment(i)) != i:
            i = end
        elif c == '"' and _token_before(source, i) in _CPP_RAW:
            paren = source.find("(", i)
            close = ")" + source[i + 1 : paren] + '"'
            end = source.find(close, paren)
            end = len(source) if paren < 0 or end < 0 else end + len(close)
            code.blank(i, end)
            i = end
        elif c == "'" and _token_before(source, i)[:1].isdigit():
            i += 1
        elif c in "\"'":
            i = code.quoted(i, escapes=True)
        else:
            i += 1
    return code.text()


def _regex_end(source: str, i: int) -> int | None:
    """Returns the index after the regex literal that the slash at i opens, or None
    when the line ends first."""
    j, in_class = i + 1, False
    while j < len(source):
        c = source[j]
        if c == "\n":
            return None
        if c == "\\":
            j += 2
            continue
        if c == "[":
            in_class = True
        elif c == "]":
            in_class = False
        elif c == "/" and not in_class:
            j += 1
            while j < len(source) and source[j].isalpha():
                j += 1
            return j
        j += 1
    return None


class _TypeScript:
    """A lexer for TypeScript that knows regex literals and nested template literals."""

    def __init__(self, source: str) -> None:
        self.source, self.code = source, _Code(source)
        # depth counts the open braces. Each entry of subs is the depth that the
        # closing brace of a template substitution returns to.
        self.depth, self.subs = 0, list[int]()

    def template(self, start: int) -> int:
        """Makes template text from start blank, up to the closing backtick or the next
        substitution, and returns the index after it. The backticks stay, so a tagged
        template still reads as a call."""
        s, j = self.source, start
        while j < len(s):
            if s[j] == "\\":
                j += 2
            elif s[j] == "`":
                self.code.blank(start, j)
                return j + 1
            elif s.startswith("${", j):
                self.code.blank(start, j)
                self.subs.append(self.depth)
                self.depth += 1
                return j + 2
            else:
                j += 1
        self.code.blank(start, len(s))
        return len(s)

    def strip(self) -> str:
        s, i, last, word = self.source, 0, "", ""
        while i < len(s):
            c = s[i]
            if (end := self.code.comment(i)) != i:
                i = end
                continue
            regex = c == "/" and (
                last in _REGEX_AFTER or last == "" or word in _REGEX_KEYWORDS
            )
            if regex and (stop := _regex_end(s, i)) is not None:
                self.code.blank(i, stop)
                end = stop
            elif c in "\"'":
                end = self.code.quoted(i, escapes=True)
            elif c == "`":
                end = self.template(i + 1)
            elif c == "}" and self.subs and self.depth - 1 == self.subs[-1]:
                self.subs.pop()
                self.depth -= 1
                end = self.template(i + 1)
            else:
                self.depth += {"{": 1, "}": -1}.get(c, 0)
                if c.isalnum() or c in "_$":
                    word = word + c if last.isalnum() or last in "_$" else c
                elif not c.isspace():
                    word = ""
                if not c.isspace():
                    last = c
                i += 1
                continue
            i, last, word = end, '"', ""
        return self.code.text()


def _strip_ts(source: str) -> str:
    return _TypeScript(source).strip()


def _strip_py(source: str) -> str:
    """Returns source with Python comments and strings made blank. It raises
    tokenize.TokenError for a file that does not tokenize."""
    starts = [0]
    for line in io.StringIO(source).readlines():
        starts.append(starts[-1] + len(line))
    blanked = {tokenize.COMMENT, tokenize.STRING}
    blanked |= {
        getattr(tokenize, kind)
        for part in ("START", "MIDDLE", "END")
        for kind in (f"FSTRING_{part}", f"TSTRING_{part}")
        if hasattr(tokenize, kind)
    }
    code = _Code(source)
    for t in tokenize.generate_tokens(io.StringIO(source).readline):
        if t.type in blanked:
            start = starts[t.start[0] - 1] + t.start[1]
            code.blank(start, starts[t.end[0] - 1] + t.end[1])
    return code.text()


_LANGUAGES: dict[str, tuple[Callable[[str], str], tuple[re.Pattern[str], ...]]] = {
    "go": (_strip_go, _GO),
    "ts": (_strip_ts, _TS),
    "py": (_strip_py, _PY),
    "integration": (_strip_py, _INTEGRATION),
    "cpp": (_strip_cpp, _CPP),
}


def markers(source: str, lang: str) -> dict[int, list[str]]:
    """Returns the skip markers in source by line, after comments and strings. It
    raises KeyError for a language that it does not know."""
    strip, patterns = _LANGUAGES[lang]
    found: dict[int, list[str]] = {}
    for number, line in enumerate(strip(source).split("\n"), start=1):
        hits = [m.group(0).strip() for p in patterns for m in p.finditer(line)]
        if hits:
            found[number] = hits
    return found


def _git(root: Path, *args: str) -> str:
    out = subprocess.run(["git", *args], cwd=root, check=True, stdout=subprocess.PIPE)
    return out.stdout.decode("utf-8")


def _changed(root: Path, base: str) -> list[tuple[str | None, str | None]]:
    """Returns each test file that HEAD adds, changes, or removes since base, as its
    path at base and its path at HEAD, None where the file is not there."""
    diff = _git(root, "diff", "--name-status", "-z", "-M", "-l0", base, "HEAD")
    fields, changed, i = diff.split("\0"), list[tuple[str | None, str | None]](), 0
    while i < len(fields) and fields[i]:
        status = fields[i][0]
        if status in "RC":
            old, new, i = fields[i + 1], fields[i + 2], i + 3
        else:
            old, new, i = fields[i + 1], fields[i + 1], i + 2
        pair = (None if status in "AC" else old, None if status == "D" else new)
        if any(p is not None and language(p) is not None for p in pair):
            changed.append(pair)
    return changed


def _read(root: Path, rev: str, path: str | None) -> tuple[str, dict[int, list[str]]]:
    """Returns the text of the test file at path in rev and its markers, or nothing for
    a path that is None or names no test file."""
    if path is None or (lang := language(path)) is None:
        return "", {}
    text = _git(root, "show", f"{rev}:{path}")
    return text, markers(text, lang)


def check(*, root: Path, base: str) -> list[Finding]:
    """Returns a finding for each added line with a marker when the sum of the markers
    in the test files that HEAD touches rose since base."""
    total_was = total_now = 0
    added: list[tuple[str, int, list[str]]] = []
    rose: list[str] = []
    for old, new in _changed(root, base):
        before, was = _read(root, base, old)
        after, now = _read(root, "HEAD", new)
        count_was, count_now = sum(map(len, was.values())), sum(map(len, now.values()))
        total_was, total_now = total_was + count_was, total_now + count_now
        if new is None or not now:
            continue
        if count_now > count_was:
            rose.append(new)
        matcher = difflib.SequenceMatcher(
            None, before.split("\n"), after.split("\n"), autojunk=False
        )
        added += [
            (new, line, now[line])
            for tag, _, _, j1, j2 in matcher.get_opcodes()
            if tag in ("insert", "replace")
            for line in range(j1 + 1, j2 + 1)
            if line in now
        ]
    if total_now <= total_was:
        return []
    note = (
        f"skip markers in the changed test files: {total_was} before, {total_now} now"
    )
    return [
        Finding(
            path=path,
            line=line,
            message=f"adds a skipped test: `{', '.join(hits)}` ({note})",
        )
        for path, line, hits in added
    ] or [
        Finding(path=path, line=0, message=f"adds a skipped test ({note})")
        for path in rose
    ]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--base", required=True, help="the commit to compare with")
    args = parser.parse_args()
    if not args.base:
        parser.error("--base names no commit")
    findings = check(root=Path(), base=args.base)
    for f in findings:
        print(f"{f.path}:{f.line}: {f.message}")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
