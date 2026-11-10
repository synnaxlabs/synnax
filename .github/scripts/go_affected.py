#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the directories of the packages in the Go module of the working directory
that the Go files in CHANGED reach. A test file reaches only its own package.

    CHANGED="<paths from the repository root>" python3 go_affected.py
"""

import os
import re
import subprocess
from collections.abc import Iterable
from pathlib import Path

# go list -test names a package recompiled for a test binary "p [p.test]".
_VARIANT = re.compile(r" \[.*\]$")


def _list(template: str) -> list[list[str]]:
    out = subprocess.run(
        ["go", "list", "-e", "-test", "-deps", "-f", template, "./..."],
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    return [line.split("\t") for line in out.splitlines() if line]


def affected(
    changed: Iterable[Path],
    module: Path,
    packages: list[tuple[str, str, list[str]]],
) -> list[str]:
    """Returns the sorted directories of the packages in module that a change reaches.
    packages holds the import path, directory, and dependencies of each package."""
    changed = list(changed)
    own = {p.parent for p in changed}
    shared = {p.parent for p in changed if not p.name.endswith("_test.go")}
    hit = {_VARIANT.sub("", path) for path, d, _ in packages if Path(d) in shared}
    out = set()
    for path, d, deps in packages:
        if not d or not Path(d).is_relative_to(module) or path.endswith(".test"):
            continue
        names = {_VARIANT.sub("", n) for n in [path, *deps]}
        if Path(d) in own or names & hit:
            rel = Path(d).relative_to(module).as_posix()
            out.add("." if rel == "." else f"./{rel}")
    return sorted(out)


def main() -> None:
    changed = os.environ["CHANGED"].split()
    root = Path(
        subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True,
            text=True,
            check=True,
        ).stdout.strip()
    )
    rows = _list('{{.ImportPath}}\t{{.Dir}}\t{{join .Deps " "}}')
    packages = [(r[0], r[1], r[2].split() if len(r) > 2 else []) for r in rows]
    print(" ".join(affected((root / c for c in changed), Path.cwd(), packages)))


if __name__ == "__main__":
    main()
