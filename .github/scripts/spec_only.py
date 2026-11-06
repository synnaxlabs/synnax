#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints whether a change touches only unit tests.

    python3 spec_only.py <base>

Compares the base commit with HEAD and prints true when every changed file is a unit
test outside integration/, else false. No unit test is part of a build, so a merge
group that changes only unit tests skips integration. An empty base or an empty diff
prints false.
"""

import subprocess
import sys
from pathlib import PurePosixPath

SUFFIXES = ("_test.go", "_test.cpp", ".spec.ts", ".spec.tsx")


def is_spec(path: str) -> bool:
    """Returns True if path is a unit test outside integration/."""
    if path.startswith("integration/"):
        return False
    name = PurePosixPath(path).name
    return name.endswith(SUFFIXES) or (
        name.startswith("test_") and name.endswith(".py")
    )


def spec_only(paths: list[str]) -> bool:
    """Returns True if paths is not empty and holds only unit tests."""
    return len(paths) > 0 and all(is_spec(p) for p in paths)


def changed(base: str) -> list[str]:
    """Returns the files changed between base and HEAD. A rename lists both names."""
    return subprocess.run(
        ["git", "diff", "--name-only", "--no-renames", base, "HEAD"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout.splitlines()


def main(argv: list[str]) -> None:
    base = argv[1]
    if not base:
        print("false")
        return
    subprocess.run(
        ["git", "fetch", "--no-tags", "--depth=1", "origin", base], check=True
    )
    print(str(spec_only(changed(base))).lower())


if __name__ == "__main__":
    main(sys.argv)
