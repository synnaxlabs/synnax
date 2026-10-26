#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the commands, flags, and config keys of the Driver, one sorted line each.

Compare the output of two revisions to see if a pull request changes them. The script
reads the C++ source as text, so it needs no build. It reports:

- each subcommand and flag the Driver command line accepts;
- each key a parser reads in the files that load the Driver config (config file,
  environment, and arguments);
- each member, with its default, of a config struct that has an `override` method.

    uv run --project scripts scripts/surface_driver.py .
"""

import argparse
import re
import sys
from pathlib import Path

SOURCES = [
    "client/cpp/synnax.h",
    "driver/cmd/*.cpp",
    "driver/common/sample_clock.h",
    "driver/main.cpp",
    "driver/rack/*.cpp",
    "driver/rack/rack.h",
    "driver/task/task.h",
]

COMMENT = re.compile(r"//[^\n]*|/\*.*?\*/", re.DOTALL)
COMMAND = re.compile(r'\bcommand == "([\w-]+)"')
READ = re.compile(
    r"\.(field|flag|optional|required|optional_child|child)\s*(?:<[^()]*?>)?\s*"
    r'\(\s*"([^"]+)"(?:\s*,\s*"(-[^"]*)")?'
)
STRUCT = re.compile(r"\bstruct\s+(\w+)\s*(?::[^{;]*)?\{")


def files(root: Path) -> list[Path]:
    out = {p for pattern in SOURCES for p in root.glob(pattern)}
    return sorted(p for p in out if not p.stem.endswith("_test"))


def body(src: str, open_brace: int) -> str:
    depth = 0
    for i in range(open_brace, len(src)):
        if src[i] == "{":
            depth += 1
        elif src[i] == "}":
            depth -= 1
            if depth == 0:
                return src[open_brace + 1 : i]
    raise ValueError("unbalanced braces")


def members(struct_body: str) -> list[str]:
    top, depth = [], 0
    for ch in struct_body:
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            top.append(";")
        elif depth == 0:
            top.append(ch)
    out = []
    for decl in "".join(top).split(";"):
        decl = " ".join(decl.split())
        decl = re.sub(r"^(public|private|protected):\s*", "", decl)
        if (
            decl
            and "(" not in decl.split("=")[0]
            and not decl.startswith(("template", "friend", "using", "static", "#"))
        ):
            out.append(decl)
    return out


def report(root: Path) -> str:
    lines: set[str] = set()
    for path in files(root):
        rel = path.relative_to(root).as_posix()
        src = COMMENT.sub("", path.read_text())
        lines.update(f"{rel} command {m}" for m in COMMAND.findall(src))
        for method, key, short in READ.findall(src):
            lines.add(f"{rel} {method} {key}" + (f" {short}" if short else ""))
        for m in STRUCT.finditer(src):
            struct_body = body(src, m.end() - 1)
            if "override(" not in struct_body:
                continue
            lines.update(f"{rel} {m.group(1)} member {d}" for d in members(struct_body))
    return "".join(f"{line}\n" for line in sorted(lines))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("root", type=Path, help="repository root")
    sys.stdout.write(report(parser.parse_args().root))


if __name__ == "__main__":
    main()
