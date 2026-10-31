#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports TypeScript exports that only their own module or test code use.

Knip in production mode reports each export that no production file uses. This check
traces each finding with Knip's `--trace-export` and reads which members each importing
file references. It drops a finding that a file outside the export's module uses, when
that file is not test code, or when the export lives in a test utility module. A module
is the nearest directory with an index file. Test code is a spec, test, or bench file,
or a file under a test utility directory: `testutil`, `test`, or `mock`.

    python3 scripts/check_exports_ts.py
"""

import json
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path

KNIP = "node_modules/.bin/knip"
INDEX_FILES = ("index.ts", "index.tsx")
TEST_DIRS = {"testutil", "test", "mock"}
TEST_SUFFIXES = (".spec", ".test", ".bench")

_ROOT = re.compile(r"^(?P<file>[^\s│├└]\S*?):(?P<name>[^\s\[]+)$")
_NODE = re.compile(
    r"^(?P<prefix>[│ ]*)[├└]── (?P<file>\S+?):(?P<kind>\w+)\[(?P<name>[^\]]*)\]"
)
_REFS = re.compile(r"^[│ ]*refs: \[(?P<refs>.*)\]$")


@dataclass(frozen=True, kw_only=True)
class Finding:
    """An export that only the specs of its own module use."""

    path: str
    line: int
    name: str

    def __str__(self) -> str:
        return (
            f"{self.path}:{self.line}: {self.name} is exported but only its own module "
            "or test code uses it"
        )


@dataclass
class _Import:
    export: tuple[str, str]
    file: str
    imported: str
    qualified: str
    refs: list[str] = field(default_factory=list)

    def uses(self) -> bool:
        if self.imported == self.qualified:
            return True
        prefix = f"{self.qualified}."
        return any(r == self.qualified or r.startswith(prefix) for r in self.refs)


def users(trace: str) -> dict[tuple[str, str], set[str]]:
    """Returns the files that use each export in a Knip trace, keyed by (file, name).

    A file that imports a namespace uses the export only when it references the
    export's member of that namespace.
    """
    result: dict[tuple[str, str], set[str]] = {}
    imports: list[_Import] = []
    export: tuple[str, str] | None = None
    names: list[str] = []
    for line in trace.splitlines():
        if m := _ROOT.match(line):
            export = (m["file"], m["name"])
            result[export] = set()
            names = [m["name"]]
        elif export is None:
            continue
        elif m := _NODE.match(line):
            depth = len(m["prefix"]) // 4 + 1
            names = names[:depth]
            if m["kind"] == "import":
                imports.append(_Import(export, m["file"], m["name"], names[-1]))
            names.append(m["name"])
        elif (m := _REFS.match(line)) and imports:
            imports[-1].refs = [r.strip() for r in m["refs"].split(",")]
    for imp in imports:
        if imp.uses():
            result[imp.export].add(imp.file)
    return result


def is_test_utility(file: str) -> bool:
    """Reports whether file lives under a test utility directory."""
    return any(part in TEST_DIRS for part in Path(file).parent.parts)


def is_test_code(file: str) -> bool:
    """Reports whether file is a spec, test, or bench file or a test utility."""
    name = Path(file).name
    return is_test_utility(file) or any(f"{s}." in name for s in TEST_SUFFIXES)


def module_dir(repo: Path, *, file: str) -> Path:
    """Returns the directory of the module that holds file."""
    path = repo / file
    for parent in path.parents:
        if any((parent / index).exists() for index in INDEX_FILES):
            return parent
        if parent.name == "src" or parent == repo:
            break
    return path.parent


def knip(repo: Path, *args: str) -> str:
    """Runs Knip and returns its output. Knip exits with 1 when it reports issues.

    :raises RuntimeError: If Knip exits with any other status.
    """
    out = subprocess.run(
        [KNIP, *args], cwd=repo, capture_output=True, text=True, check=False
    )
    if out.returncode not in (0, 1):
        raise RuntimeError(f"knip {' '.join(args)} failed:\n{out.stderr}")
    return out.stdout


def candidates(repo: Path) -> list[Finding]:
    """Returns the exports that Knip in production mode reports."""
    out = knip(repo, "--production", "--include", "exports,types", "--reporter", "json")
    report = json.loads(out[out.index('{"issues"') :].splitlines()[0])
    return [
        Finding(path=issue["file"], line=item["line"], name=item["name"])
        for issue in report["issues"]
        for kind in ("exports", "types")
        for item in issue.get(kind, [])
    ]


def check(repo: Path) -> list[Finding]:
    """Returns the exports that only their own module or test code use."""
    found = candidates(repo)
    used: dict[tuple[str, str], set[str]] = {}
    for name in sorted({f.name for f in found}):
        used |= users(knip(repo, "--trace-export", name))
    result = []
    for f in found:
        own = module_dir(repo, file=f.path)
        outside = [
            u
            for u in used.get((f.path, f.name), set())
            if module_dir(repo, file=u) != own
        ]
        if is_test_utility(f.path) and outside:
            continue
        if all(is_test_code(u) for u in outside):
            result.append(f)
    return sorted(result, key=lambda f: (f.path, f.line))


def main() -> None:
    repo = Path.cwd()
    findings = check(repo)
    for f in findings:
        print(f)
    if findings:
        print(
            f"{len(findings)} exports only their own module or test code use",
            file=sys.stderr,
        )
        sys.exit(1)


if __name__ == "__main__":
    main()
