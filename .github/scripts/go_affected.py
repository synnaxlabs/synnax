#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the Go modules that a change affects: each module that holds a changed file,
and each module whose go.mod replaces an affected module with a local path.

Reads the changed paths, relative to the repository root, from stdin, one per line.
Prints the affected modules among the given ones, in the given order, one per line.

Usage: go_affected.py <module dir>...
"""

import posixpath
import re
import sys
from collections.abc import Iterable
from pathlib import Path

REPLACE = re.compile(r"=>\s*(\.\.?/\S*)")


def local_replaces(root: Path, module: str) -> set[str]:
    """Returns the module directories, relative to root, that the go.mod of module
    replaces with a local path."""
    text = (root / module / "go.mod").read_text()
    return {posixpath.normpath(f"{module}/{t}") for t in REPLACE.findall(text)}


def affected(root: Path, modules: list[str], changed: Iterable[str]) -> list[str]:
    """Returns the modules in the given order that hold a changed path or that depend
    on such a module through a local replace."""
    dependents: dict[str, set[str]] = {m: set() for m in modules}
    for m in modules:
        for target in local_replaces(root, m):
            if target in dependents:
                dependents[target].add(m)
    paths = list(changed)
    pending = [m for m in modules if any(p.startswith(f"{m}/") for p in paths)]
    found = set(pending)
    while pending:
        for d in dependents[pending.pop()] - found:
            found.add(d)
            pending.append(d)
    return [m for m in modules if m in found]


def main() -> None:
    changed = [line.strip() for line in sys.stdin if line.strip()]
    for m in affected(Path("."), sys.argv[1:], changed):
        print(m)


if __name__ == "__main__":
    main()
