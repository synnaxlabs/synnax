#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the public API of the C++ client, one line per declaration.

Compare the output of two revisions to see if a pull request changes the API. The script
reads the headers as text, so it needs no build. It reports each public declaration
with its enclosing scope, in source order, because the order of members and enum entries
is part of the API. It leaves out function bodies, constructor initializers, private
members, and the protobuf and JSON codecs that Oracle generates.

    uv run --project scripts scripts/surface_cpp.py .
"""

import argparse
import re
import sys
from pathlib import Path

HEADERS = "client/cpp/**/*.h"
EXCLUDED_DIR = "testutil"
EXCLUDED_SUFFIXES = ("proto.gen.h", "json.gen.h")

# Strings come first, so that a comment marker inside one is left alone.
TEXT = re.compile(
    r'"(?:\\.|[^"\\\n])*"|\'(?:\\.|[^\'\\\n])*\'|//[^\n]*|/\*.*?\*/', re.DOTALL
)
DIRECTIVE = re.compile(r"^[ \t]*#.*(?:\\\n.*)*$", re.MULTILINE)
ACCESS = re.compile(r"\s*(public|private|protected)\s*:(?!:)")
NAMESPACE = re.compile(r"(?:inline\s+)?namespace(?:\s+([\w:]+))?\s*$")
TYPE = re.compile(r"(?:template\s*<.*>\s*)?(class|struct|union|enum)\b(?:\s+class\b)?")
TYPE_NAME = re.compile(r"(\w+)\s*(?:final\s*)?(?::(?!:).*)?$")
FORWARD = re.compile(r"(?:class|struct) \w+$")
# Space that only a line wrap puts there, so that a reformat changes no line.
WRAP = re.compile(r"(?:(?<=[(<{])|(?<=::)) | (?=[)>}])")
STRING = re.compile(r'("(?:\\.|[^"\\\n])*"|\'(?:\\.|[^\'\\\n])*\')')
SPACE = re.compile(r"\s+")
SKIPPED = ("using namespace ", "static_assert")
FRIEND_TYPE = re.compile(r"friend (?:class|struct) ")


def strip(src: str) -> str:
    comments = TEXT.sub(lambda m: " " if m[0][0] == "/" else m[0], src)
    return DIRECTIVE.sub("", comments)


def closing(src: str, *, start: int) -> int:
    opener, closer = src[start], {"{": "}", "(": ")"}[src[start]]
    depth = 0
    i = start
    while i < len(src):
        if src[i] in "\"'":
            i = after_text(src, start=i)
            continue
        if src[i] == opener:
            depth += 1
        elif src[i] == closer:
            depth -= 1
            if depth == 0:
                return i
        i += 1
    raise ValueError("unbalanced braces")


def normalize(text: str) -> str:
    # re.split puts the literals at the odd indices.
    parts = STRING.split(text)
    parts[::2] = [WRAP.sub("", SPACE.sub(" ", part)) for part in parts[::2]]
    return "".join(parts).strip()


def after_text(src: str, *, start: int) -> int:
    match = TEXT.match(src, start)
    if match is None:
        raise ValueError(f"unterminated literal at offset {start}")
    return match.end()


def signature(head: str) -> tuple[str, bool]:
    """Returns the head without a constructor initializer, and if it had one."""
    depth = 0
    seen = False
    for i, ch in enumerate(head):
        if ch == "(":
            depth += 1
            seen = True
        elif ch == ")":
            depth -= 1
        elif (
            seen
            and depth == 0
            and ch == ":"
            and head[i - 1] != ":"
            and head[i + 1 : i + 2] != ":"
        ):
            return head[:i], True
    return head, False


def skip_initializers(src: str, *, start: int) -> int:
    """Skips the rest of a constructor after its first brace pair: a brace initializer
    ends one pair, so the body can sit behind more of them."""
    i = start
    while True:
        rest = src[i:].lstrip()
        if not rest or rest[0] not in ",{":
            return i
        while src[i] != "{":
            if src[i] == "(":
                i = closing(src, start=i)
            elif src[i] in "\"'":
                i = after_text(src, start=i) - 1
            i += 1
        i = closing(src, start=i) + 1


def scan(src: str, *, scope: str, public: bool, out: list[str]) -> None:
    start = i = paren = 0
    while i < len(src):
        access = ACCESS.match(src, start) if i == start else None
        if access is not None:
            public = access[1] == "public"
            start = i = access.end()
            continue
        ch = src[i]
        if ch in "\"'":
            i = after_text(src, start=i)
            continue
        if ch in "()":
            paren += 1 if ch == "(" else -1
        elif ch == "{" and paren > 0:
            i = closing(src, start=i)
        elif ch == ";":
            declare(src[start:i], scope=scope, public=public, out=out)
            start = i + 1
        elif ch == "{":
            end = closing(src, start=i)
            i = enter(
                src,
                open_brace=i,
                end=end,
                head=src[start:i],
                scope=scope,
                public=public,
                out=out,
            )
            start = i
            continue
        elif ch.isspace() and i == start:
            start = i + 1
        i += 1


def enter(
    src: str,
    *,
    open_brace: int,
    end: int,
    head: str,
    scope: str,
    public: bool,
    out: list[str],
) -> int:
    head = normalize(head)
    body = src[open_brace + 1 : end]
    namespace = NAMESPACE.fullmatch(head)
    if namespace is not None:
        if namespace[1] is not None:
            scan(body, scope=f"{scope}{namespace[1]}::", public=True, out=out)
        return end + 1
    kind = TYPE.match(head)
    if kind is not None and "(" not in head:
        declare(f"{head} {{}}", scope=scope, public=public, out=out)
        if not public:
            return end + 1
        name = TYPE_NAME.search(head)
        inner = f"{scope}{name[1] if name else ''}::"
        if kind[1] == "enum":
            for entry in body.split(","):
                declare(entry, scope=inner, public=True, out=out)
        else:
            scan(body, scope=inner, public=kind[1] != "class", out=out)
        return end + 1
    if "(" not in head:
        declare(f"{head}{{{body}}}", scope=scope, public=public, out=out)
        return end + 1
    declaration, initialized = signature(head)
    declare(declaration, scope=scope, public=public, out=out)
    return skip_initializers(src, start=end + 1) if initialized else end + 1


def declare(text: str, *, scope: str, public: bool, out: list[str]) -> None:
    line = normalize(text)
    # A friend function is callable from anywhere, whichever section declares it.
    if not (public or line.startswith("friend ")):
        return
    if FORWARD.fullmatch(line) or FRIEND_TYPE.match(line) or line.startswith(SKIPPED):
        return
    if line:
        out.append(f"{scope} {line}" if scope else line)


def files(root: Path) -> list[Path]:
    return sorted(
        p
        for p in root.glob(HEADERS)
        if EXCLUDED_DIR not in p.parts and not p.name.endswith(EXCLUDED_SUFFIXES)
    )


def report(root: Path) -> str:
    lines: list[str] = []
    for path in files(root):
        out: list[str] = []
        scan(strip(path.read_text()), scope="", public=True, out=out)
        rel = path.relative_to(root).as_posix()
        lines += [f"{rel} {line}" for line in out]
    return "".join(f"{line}\n" for line in lines)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("root", type=Path, help="repository root")
    sys.stdout.write(report(parser.parse_args().root))


if __name__ == "__main__":
    main()
