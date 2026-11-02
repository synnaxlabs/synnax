#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Parses the phase table at the end of an RFC and prints it as JSON.

The table format is in docs/tech/rfc/AGENTS.md. Exits non-zero with every error
found when the table is missing or malformed.

Usage: phase_table.py <rfc.md>
"""

import json
import re
import sys
from dataclasses import asdict, dataclass
from pathlib import Path

HEADER = ("Phase", "Scope", "Touches", "Depends on")
MAX_SCOPE_LENGTH = 60
_PATH = re.compile(r"`([^`]+)`")
_CELL_SPLIT = re.compile(r"(?<!\\)\|")
_SEPARATOR = re.compile(r":?-+:?")


@dataclass(frozen=True, kw_only=True)
class Phase:
    number: int
    scope: str
    touches: list[str]
    depends_on: list[int]


def _cells(line: str) -> list[str]:
    inner = line.strip().removeprefix("|").removesuffix("|")
    return [c.strip().replace(r"\|", "|") for c in _CELL_SPLIT.split(inner)]


def _parse_row(cells: list[str], expected: int) -> tuple[Phase | None, list[str]]:
    errors: list[str] = []
    if len(cells) != len(HEADER):
        return None, [f"row {expected}: want {len(HEADER)} cells, got {len(cells)}"]
    number_cell, scope, touches_cell, depends_cell = cells
    if number_cell != str(expected):
        errors.append(f"row {expected}: phase is {number_cell!r}, want {expected}")
    if not scope:
        errors.append(f"phase {expected}: scope is empty")
    elif len(scope) > MAX_SCOPE_LENGTH:
        errors.append(f"phase {expected}: scope is over {MAX_SCOPE_LENGTH} characters")
    elif scope.endswith("."):
        errors.append(f"phase {expected}: scope ends in a period")
    touches = [p.strip() for p in touches_cell.split(",")]
    if not all(_PATH.fullmatch(p) for p in touches):
        errors.append(f"phase {expected}: touches must be backticked paths")
    depends_on: list[int] = []
    if depends_cell != "-":
        for dep in (d.strip() for d in depends_cell.split(",")):
            if not dep.isdigit() or not 1 <= int(dep) < expected:
                errors.append(f"phase {expected}: {dep!r} is not an earlier phase")
            else:
                depends_on.append(int(dep))
    paths = [m.group(1) for p in touches if (m := _PATH.fullmatch(p))]
    return Phase(
        number=expected, scope=scope, touches=paths, depends_on=depends_on
    ), errors


def parse(markdown: str) -> list[Phase]:
    """Returns the phases of the one phase table in markdown.

    Raises ValueError listing every problem when there is not exactly one phase
    table or any row breaks the format.
    """
    lines = markdown.splitlines()
    starts = [i for i, line in enumerate(lines) if tuple(_cells(line)) == HEADER]
    if len(starts) != 1:
        raise ValueError(f"want one phase table, found {len(starts)}")
    separator = lines[starts[0] + 1] if starts[0] + 1 < len(lines) else ""
    cells = _cells(separator)
    if len(cells) != len(HEADER) or not all(_SEPARATOR.fullmatch(c) for c in cells):
        raise ValueError("the phase table header has no separator row")
    rows: list[list[str]] = []
    for line in lines[starts[0] + 2 :]:
        if not line.strip().startswith("|"):
            break
        rows.append(_cells(line))
    if not rows:
        raise ValueError("the phase table has no rows")
    phases: list[Phase] = []
    errors: list[str] = []
    for i, cells in enumerate(rows, start=1):
        phase, row_errors = _parse_row(cells, i)
        errors.extend(row_errors)
        if phase is not None:
            phases.append(phase)
    if errors:
        raise ValueError("\n".join(errors))
    return phases


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: phase_table.py <rfc.md>")
    try:
        phases = parse(Path(sys.argv[1]).read_text())
    except ValueError as e:
        sys.exit(f"{sys.argv[1]}:\n{e}")
    print(json.dumps([asdict(p) for p in phases], indent=2))


if __name__ == "__main__":
    main()
