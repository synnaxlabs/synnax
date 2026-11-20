#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Fails when a Linux binary cannot load on glibc FLOOR, the oldest the Driver supports.

Usage: check_glibc_floor.py <binary>

A binary must not need a GLIBC_ symbol version newer than FLOOR, and must link libstdc++
and libgcc statically, because NI Linux Real-Time ships older copies of both.
"""

import re
import subprocess
import sys

FLOOR = (2, 35)
GLIBC = re.compile(r"\bGLIBC_(\d+(?:\.\d+)+)\b")
NEEDED = re.compile(r"^\s*NEEDED\s+(\S+)", re.MULTILINE)
SHARED = ("libstdc++.so", "libgcc_s.so")


def problems(symbols: str, headers: str) -> list[str]:
    """Returns why a binary with the given objdump -T and -p output cannot load."""
    found = []
    versions = {tuple(map(int, v.split("."))) for v in GLIBC.findall(symbols)}
    if newer := sorted(v for v in versions if v > FLOOR):
        names = ", ".join("GLIBC_" + ".".join(map(str, v)) for v in newer)
        found.append(f"needs {names}, newer than the glibc {FLOOR[0]}.{FLOOR[1]} floor")
    found.extend(
        f"links {library} dynamically"
        for library in NEEDED.findall(headers)
        if library.startswith(SHARED)
    )
    return found


def main(binary: str) -> int:
    def objdump(flag: str) -> str:
        return subprocess.check_output(["objdump", flag, binary], text=True)

    found = problems(objdump("-T"), objdump("-p"))
    for problem in found:
        print(f"{binary} {problem}", file=sys.stderr)
    return 1 if found else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
