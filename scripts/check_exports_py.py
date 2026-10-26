#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reports public Python names that no other module imports.

A top-level name without a leading underscore is wrongly public when no other module
imports it, except the tests of its own package. Rename it with a leading underscore.
A name stays public when an `__init__.py` re-exports it, when a public name used
elsewhere carries it in an annotation or a base class, or when a `pyproject.toml`
entry point names it. The check skips generated files and `__init__.py` definitions.

    uv run --project scripts scripts/check_exports_py.py
"""

import argparse
import ast
import sys
import tomllib
import warnings
from collections.abc import Iterator
from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True)
class Package:
    """A distribution whose modules the check reads."""

    root: str
    """The directory that holds the import roots, relative to the repository."""
    checked: tuple[str, ...]
    """The import roots whose names the check reports."""
    tests: tuple[str, ...] = ("tests",)
    """The directories whose imports do not keep a name of this package public."""


PACKAGES = (
    Package("alamos/py", ("alamos",)),
    Package("client/py", ("synnax",)),
    Package("freighter/py", ("freighter",)),
    Package("integration", ("console", "framework"), ("framework/tests",)),
    Package("x/py", ("x",)),
)

SKIPPED_DIRS = {".venv", "node_modules", "build", "__pycache__"}


@dataclass
class Module:
    name: str
    path: Path
    package: Package
    test: bool
    checked: bool
    tree: ast.Module
    generated: bool
    aliases: dict[str, str] = field(default_factory=dict)


@dataclass(frozen=True)
class Finding:
    """A top-level name that is public but that no other module imports."""

    path: str
    line: int
    name: str
    tests_only: bool

    def __str__(self) -> str:
        users = (
            "only its tests import it" if self.tests_only else "no module imports it"
        )
        return f"{self.path}:{self.line}: {self.name} is public but {users}"


def module_name(root: Path, path: Path) -> str:
    parts = list(path.relative_to(root).with_suffix("").parts)
    if parts[-1] == "__init__":
        parts.pop()
    return ".".join(parts)


def is_generated(path: Path, source: str) -> bool:
    return (
        path.name.endswith(("_gen.py", "_pb2.py", "_pb2_grpc.py"))
        or "Code generated" in source[:500]
    )


def parse(source: str, path: Path) -> ast.Module:
    # A bad escape in a string is a warning, not an error, and is not this check's.
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", SyntaxWarning)
        return ast.parse(source, str(path))


def load(repo: Path, packages: tuple[Package, ...]) -> list[Module]:
    modules = []
    for pkg in packages:
        root = repo / pkg.root
        for path in sorted(root.rglob("*.py")):
            rel = path.relative_to(root)
            if SKIPPED_DIRS & set(rel.parts):
                continue
            test = any(rel.is_relative_to(t) for t in pkg.tests)
            source = path.read_text()
            modules.append(
                Module(
                    name=module_name(root, path),
                    path=path,
                    package=pkg,
                    test=test,
                    checked=not test and rel.parts[0] in pkg.checked,
                    tree=parse(source, path),
                    generated=is_generated(path, source),
                )
            )
    return modules


def top_level_names(node: ast.stmt) -> list[str]:
    if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef | ast.ClassDef):
        return [node.name]
    if isinstance(node, ast.Assign):
        return [t.id for t in node.targets if isinstance(t, ast.Name)]
    if isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
        return [node.target.id]
    if isinstance(node, ast.TypeAlias):
        return [node.name.id]
    return []


def resolve(mod: Module, level: int, target: str | None) -> str:
    if level == 0:
        return target or ""
    base = mod.name.split(".")
    if mod.path.name != "__init__.py":
        base.pop()
    base = base[: len(base) - level + 1]
    return ".".join(base + ([target] if target else []))


def imports(mod: Module, known: set[str]) -> Iterator[tuple[str, str]]:
    """Yields each (module, name) pair that mod reads from another module."""
    for node in ast.walk(mod.tree):
        if isinstance(node, ast.Import):
            for a in node.names:
                local = a.asname or a.name.split(".")[0]
                mod.aliases[local] = a.name if a.asname else local
        elif isinstance(node, ast.ImportFrom):
            source = resolve(mod, node.level, node.module)
            for a in node.names:
                if a.name == "*":
                    yield source, "*"
                elif f"{source}.{a.name}" in known:
                    mod.aliases[a.asname or a.name] = f"{source}.{a.name}"
                else:
                    yield source, a.name
    for node in ast.walk(mod.tree):
        if not isinstance(node, ast.Attribute):
            continue
        chain = []
        value: ast.expr = node
        while isinstance(value, ast.Attribute):
            chain.append(value.attr)
            value = value.value
        if not isinstance(value, ast.Name) or value.id not in mod.aliases:
            continue
        source = mod.aliases[value.id]
        for part in reversed(chain):
            if f"{source}.{part}" not in known:
                yield source, part
                break
            source = f"{source}.{part}"


def entry_points(
    repo: Path, packages: tuple[Package, ...]
) -> Iterator[tuple[str, str]]:
    for pkg in packages:
        with (repo / pkg.root / "pyproject.toml").open("rb") as f:
            scripts = tomllib.load(f).get("project", {}).get("scripts", {})
        for target in scripts.values():
            source, _, name = target.partition(":")
            yield source, name.split(".")[0]


def signature(fn: ast.FunctionDef | ast.AsyncFunctionDef) -> list[ast.expr | None]:
    args = fn.args
    params = [*args.posonlyargs, *args.args, *args.kwonlyargs]
    params += [a for a in (args.vararg, args.kwarg) if a is not None]
    return [a.annotation for a in params] + [fn.returns]


def annotation_names(node: ast.stmt) -> set[str]:
    """Returns the names that the signature, bases, or annotations of node read."""
    exprs: list[ast.expr | None] = []
    if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
        exprs += signature(node)
    elif isinstance(node, ast.ClassDef):
        exprs += node.bases
        for item in node.body:
            if isinstance(item, ast.AnnAssign):
                exprs.append(item.annotation)
            elif isinstance(item, ast.FunctionDef | ast.AsyncFunctionDef) and (
                not item.name.startswith("_") or item.name == "__init__"
            ):
                exprs += signature(item)
    elif isinstance(node, ast.AnnAssign):
        exprs.append(node.annotation)
    elif isinstance(node, ast.TypeAlias | ast.Assign):
        exprs.append(node.value)
    names = set()
    for e in exprs:
        if e is None:
            continue
        for n in ast.walk(e):
            if isinstance(n, ast.Name):
                names.add(n.id)
            elif isinstance(n, ast.Constant) and isinstance(n.value, str):
                names.update(n.value.replace("[", " ").replace("]", " ").split())
    return names


def check(repo: Path, packages: tuple[Package, ...] = PACKAGES) -> list[Finding]:
    """Returns the wrongly public names in packages, sorted by path and line."""
    modules = load(repo, packages)
    known = {m.name for m in modules if not m.test}
    defs: dict[tuple[str, str], tuple[Module, ast.stmt]] = {}
    for m in modules:
        if not m.checked or m.generated or m.path.name == "__init__.py":
            continue
        for node in m.tree.body:
            for name in top_level_names(node):
                if not name.startswith("_"):
                    defs[(m.name, name)] = (m, node)
    used: set[tuple[str, str]] = set(entry_points(repo, packages))
    tested: set[tuple[str, str]] = set()
    for m in modules:
        for source, name in imports(m, known):
            if source == m.name:
                continue
            keys = (
                [k for k in defs if k[0] == source] if name == "*" else [(source, name)]
            )
            for key in keys:
                owner = defs.get(key)
                if m.test and owner is not None and owner[0].package == m.package:
                    tested.add(key)
                else:
                    used.add(key)
    pending = [k for k in defs if k in used]
    while pending:
        mod, node = defs[pending.pop()]
        for name in annotation_names(node):
            key = (mod.name, name)
            if key in defs and key not in used:
                used.add(key)
                pending.append(key)
    findings = [
        Finding(
            path=str(m.path.relative_to(repo)),
            line=node.lineno,
            name=name,
            tests_only=(mod, name) in tested,
        )
        for (mod, name), (m, node) in defs.items()
        if (mod, name) not in used
    ]
    return sorted(findings, key=lambda f: (f.path, f.line))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument(
        "repo", nargs="?", default=".", help="repository root (default: .)"
    )
    findings = check(Path(parser.parse_args().repo).resolve())
    for f in findings:
        print(f)
    if findings:
        print(f"{len(findings)} wrongly public names", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
