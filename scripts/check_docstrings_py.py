#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports Python docstrings that use Google sections instead of Sphinx fields.

Docstrings take `:param name:`, `:returns:`, and `:raises Type:` fields. The check
reports an `Args:`, `Arguments:`, `Returns:`, `Raises:`, or `Yields:` section header,
and a `:return:` field. Ruff has no rule for either. The check skips generated files.

    python3 scripts/check_docstrings_py.py [directory ...]
"""

import argparse
import ast
import re
import sys
import warnings
from dataclasses import dataclass
from pathlib import Path

SKIPPED_DIRS = {".venv", "node_modules", "build", "dist", "__pycache__"}
_SECTION = re.compile(r"^\s*(Args|Arguments|Returns|Raises|Yields):\s*$")
_RETURN = re.compile(r"^\s*:return:")


@dataclass(frozen=True, kw_only=True)
class Finding:
    path: str
    line: int
    message: str


def _generated(source: str) -> bool:
    return any("Code generated" in line for line in source.splitlines()[:15])


def _docstrings(tree: ast.Module) -> list[ast.Constant]:
    return [
        node.value
        for node in ast.walk(tree)
        if isinstance(node, ast.Expr)
        and isinstance(node.value, ast.Constant)
        and isinstance(node.value.value, str)
    ]


def check_file(path: Path, *, root: Path) -> list[Finding]:
    """Returns the findings in one file, with paths relative to root."""
    source = path.read_text(encoding="utf-8")
    if _generated(source):
        return []
    rel = path.relative_to(root).as_posix()
    findings = []
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", SyntaxWarning)
        tree = ast.parse(source, filename=str(path))
    for doc in _docstrings(tree):
        for offset, line in enumerate(str(doc.value).splitlines()):
            if m := _SECTION.match(line):
                message = f"use Sphinx fields, not a Google {m.group(1)}: section"
            elif _RETURN.match(line):
                message = "use :returns:, not :return:"
            else:
                continue
            findings.append(
                Finding(path=rel, line=doc.lineno + offset, message=message)
            )
    return findings


def check(root: Path) -> list[Finding]:
    """Returns the findings in every Python file under root."""
    findings = []
    for path in sorted(root.rglob("*.py")):
        if SKIPPED_DIRS.isdisjoint(path.relative_to(root).parts):
            findings.extend(check_file(path, root=root))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("directories", nargs="*", type=Path, default=[Path()])
    findings = [f for d in parser.parse_args().directories for f in check(d)]
    for f in findings:
        print(f"{f.path}:{f.line}: {f.message}")
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
