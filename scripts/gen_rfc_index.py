#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Regenerates the RFC index table in docs/tech/rfc/README.md.

Reads the H1 and the Date front matter of every NNNN-slug.md in docs/tech/rfc, then
rewrites the rows between the marker comments in README.md. The prose around the table
is hand-written and is left alone.

Usage: gen_rfc_index.py [--check]
"""

import re
import sys
from pathlib import Path

RFC_DIR = Path(__file__).resolve().parent.parent / "docs" / "tech" / "rfc"
README = RFC_DIR / "README.md"
START = "<!-- begin index -->"
END = "<!-- end index -->"


def read(path: Path) -> tuple[str, str]:
    """Returns the title and date of the RFC at path."""
    text = path.read_text(encoding="utf-8")
    h1 = re.search(r"^# \d+ (.*)$", text, re.M)
    date = re.search(r"^- \*\*Date\*\*: (\d{4}-\d{2}-\d{2})$", text, re.M)
    if h1 is None:
        raise SystemExit(f"{path.name}: no '# N Title' heading")
    if date is None:
        raise SystemExit(f"{path.name}: no '- **Date**: YYYY-MM-DD' front matter")
    return h1.group(1), date.group(1)


def table() -> str:
    rows = ["| RFC | Title | Date |", "| --- | --- | --- |"]
    for path in sorted(RFC_DIR.glob("[0-9][0-9][0-9][0-9]-*.md")):
        title, date = read(path)
        rows.append(f"| {path.name[:4]} | [{title}]({path.name}) | {date} |")
    return "\n".join(rows)


def normalize(block: str) -> list[list[str]]:
    """Reduces a table to its cell text, discarding the padding Prettier adds."""
    rows = []
    for line in block.splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if all(set(c) <= {"-"} and c for c in cells):
            continue
        rows.append(cells)
    return rows


def main() -> None:
    text = README.read_text(encoding="utf-8")
    pattern = re.compile(f"{re.escape(START)}\n(?:.*?\n)?{re.escape(END)}", re.S)
    current = pattern.search(text)
    if current is None:
        raise SystemExit(f"{README}: missing {START} / {END} markers")
    fresh = table()
    if normalize(current.group(0)) == normalize(fresh):
        return
    if "--check" in sys.argv:
        raise SystemExit("RFC index is stale. Run scripts/gen_rfc_index.py.")
    README.write_text(
        pattern.sub(f"{START}\n\n{fresh}\n\n{END}", text), encoding="utf-8"
    )
    print("Wrote the RFC index. Run pnpm format to align the table.")


if __name__ == "__main__":
    main()
