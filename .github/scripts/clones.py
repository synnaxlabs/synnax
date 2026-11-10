#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Ranks the clone groups in a jscpd JSON report by the lines a fix would remove.

Usage: clones.py <jscpd-report.json> --language go|ts|python|cpp|all [--top 15]

Prints a Markdown list. The same report always gives the same list.
"""

import argparse
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

FORMATS = {
    "go": {"go"},
    "ts": {"typescript", "tsx"},
    "python": {"python"},
    "cpp": {"cpp", "c-header"},
}


@dataclass(frozen=True, order=True, kw_only=True)
class Fragment:
    path: str
    start: int
    end: int

    @property
    def lines(self) -> int:
        return self.end - self.start + 1


@dataclass(frozen=True)
class Group:
    fragments: tuple[Fragment, ...]

    @property
    def removable(self) -> int:
        """Returns the lines that go away when every copy but the longest is removed.
        Copies that overlap in one file count their shared lines once."""
        covered = {
            (f.path, line) for f in self.fragments for line in range(f.start, f.end + 1)
        }
        return len(covered) - max(f.lines for f in self.fragments)


def fragment(side: dict[str, Any]) -> Fragment:
    return Fragment(path=side["name"], start=side["start"], end=side["end"])


def groups(duplicates: list[dict[str, Any]], *, formats: set[str]) -> list[Group]:
    """Joins the clone pairs that share a fragment into groups, largest first. A group
    whose every line lies in a larger group is left out, since one fix removes both."""
    parent: dict[Fragment, Fragment] = {}

    def root(f: Fragment) -> Fragment:
        while parent.setdefault(f, f) != f:
            f = parent[f]
        return f

    for d in duplicates:
        if d["format"] in formats:
            a, b = root(fragment(d["firstFile"])), root(fragment(d["secondFile"]))
            parent[max(a, b)] = min(a, b)
    members: dict[Fragment, list[Fragment]] = {}
    for f in parent:
        members.setdefault(root(f), []).append(f)
    found = [Group(tuple(sorted(m))) for m in members.values()]
    ranked: list[Group] = []
    taken: set[tuple[str, int]] = set()
    for g in sorted(found, key=lambda g: (-g.removable, g.fragments)):
        lines = {(f.path, n) for f in g.fragments for n in range(f.start, f.end + 1)}
        if lines <= taken:
            continue
        ranked.append(g)
        taken |= lines
    return ranked


def markdown(ranked: list[Group]) -> str:
    return "\n".join(
        f"{i}. {g.removable} lines, {len(g.fragments)} copies: "
        + ", ".join(f"`{f.path}:{f.start}-{f.end}`" for f in g.fragments)
        for i, g in enumerate(ranked, 1)
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    parser.add_argument("--language", required=True, choices=[*FORMATS, "all"])
    parser.add_argument("--top", type=int, default=15)
    args = parser.parse_args()
    formats = (
        set().union(*FORMATS.values())
        if args.language == "all"
        else FORMATS[args.language]
    )
    duplicates = json.loads(args.report.read_text())["duplicates"]
    print(markdown(groups(duplicates, formats=formats)[: args.top]))


if __name__ == "__main__":
    main()
