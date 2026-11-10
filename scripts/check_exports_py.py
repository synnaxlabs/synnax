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
imports it, except test code: tests, `conftest.py`, and test utilities. A test utility
is a `testutil`, `test`, or `mock` module, a module under such a directory, or a module
of a package that holds only test utilities; tests may import its names. Rename a
finding with a leading underscore, or move a helper that tests need into a test
utility. A name stays public when an `__init__.py` re-exports it, when a public name
used elsewhere carries it in an annotation or a base class, or when a `pyproject.toml`
entry point names it. The check skips generated files and `__init__.py` definitions.

    uv run --project scripts scripts/check_exports_py.py
"""

import argparse
import ast
import itertools
import sys
import tomllib
import warnings
from collections.abc import Iterator
from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True, kw_only=True)
class Package:
    """A distribution whose modules the check reads."""

    root: str
    """The directory that holds the import roots, relative to the repository."""
    checked: tuple[str, ...]
    """The import roots whose names the check reports."""
    tests: tuple[str, ...] = ("tests",)
    """The directories that hold test code."""
    test_utility: bool = False
    """Whether every checked module is a test utility, whose names tests may import."""


PACKAGES = (
    Package(root="alamos/py", checked=("alamos",)),
    Package(root="client/py", checked=("synnax",)),
    Package(root="freighter/py", checked=("freighter",)),
    Package(
        root="integration",
        checked=("console", "framework"),
        tests=("framework/tests", "tests"),
        test_utility=True,
    ),
    Package(root="x/py", checked=("x",)),
)

SKIPPED_DIRS = {".venv", "node_modules", "build", "__pycache__"}
TEST_UTILITY_DIRS = {"testutil", "test", "mock"}


@dataclass(kw_only=True)
class Module:
    name: str
    path: Path
    package: Package
    test: bool
    """Whether the module is test code: a test, a conftest, or a test utility."""
    utility: bool
    """Whether the module is a test utility, whose names tests may import."""
    checked: bool
    tree: ast.Module
    generated: bool
    aliases: dict[str, set[str]] = field(default_factory=dict)
    """The modules each local name refers to. A name that different scopes import
    differently refers to each of them."""


@dataclass(frozen=True, kw_only=True)
class Finding:
    """A top-level name that is public but that no other module imports."""

    path: str
    line: int
    name: str
    tests_only: bool

    def __str__(self) -> str:
        users = (
            "only test code imports it" if self.tests_only else "no module imports it"
        )
        return f"{self.path}:{self.line}: {self.name} is public but {users}"


def module_name(root: Path, *, path: Path) -> str:
    parts = list(path.relative_to(root).with_suffix("").parts)
    if parts[-1] == "__init__":
        parts.pop()
    return ".".join(parts)


def is_generated(path: Path, *, source: str) -> bool:
    return (
        path.name.endswith(("_gen.py", "_pb2.py", "_pb2_grpc.py"))
        or "Code generated" in source[:500]
    )


def parse(source: str, *, path: Path) -> ast.Module:
    # A bad escape in a string is a warning, not an error, and is not this check's.
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", SyntaxWarning)
        return ast.parse(source, str(path))


def is_test_file(path: Path) -> bool:
    return path.name == "conftest.py" or path.name.startswith("test_")


def load(repo: Path, *, packages: tuple[Package, ...]) -> list[Module]:
    modules = []
    for pkg in packages:
        root = repo / pkg.root
        for path in sorted(root.rglob("*.py")):
            rel = path.relative_to(root)
            if SKIPPED_DIRS & set(rel.parts):
                continue
            in_tests = any(rel.is_relative_to(t) for t in pkg.tests)
            utility = pkg.test_utility or bool(
                TEST_UTILITY_DIRS & set(rel.with_suffix("").parts)
            )
            source = path.read_text()
            modules.append(
                Module(
                    name=module_name(root, path=path),
                    path=path,
                    package=pkg,
                    test=in_tests or utility or is_test_file(path),
                    utility=utility,
                    checked=not in_tests and rel.parts[0] in pkg.checked,
                    tree=parse(source, path=path),
                    generated=is_generated(path, source=source),
                )
            )
    return modules


def top_level_names(node: ast.stmt) -> list[str]:
    if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef | ast.ClassDef):
        return [node.name]
    if isinstance(node, ast.Assign):
        return [
            n.id
            for t in node.targets
            for n in ast.walk(t)
            if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Store)
        ]
    if isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
        return [node.target.id]
    if isinstance(node, ast.TypeAlias):
        return [node.name.id]
    return []


def resolve(mod: Module, *, level: int, target: str | None) -> str:
    if level == 0:
        return target or ""
    base = mod.name.split(".")
    if mod.path.name != "__init__.py":
        base.pop()
    base = base[: len(base) - level + 1]
    return ".".join(base + ([target] if target else []))


def imports(mod: Module, *, known: set[str]) -> Iterator[tuple[str, str]]:
    """Yields each (module, name) pair that mod reads from another module."""
    for node in ast.walk(mod.tree):
        if isinstance(node, ast.Import):
            for a in node.names:
                local = a.asname or a.name.split(".")[0]
                target = a.name if a.asname else local
                mod.aliases.setdefault(local, set()).add(target)
        elif isinstance(node, ast.ImportFrom):
            source = resolve(mod, level=node.level, target=node.module)
            for a in node.names:
                if a.name == "*":
                    yield source, "*"
                elif f"{source}.{a.name}" in known:
                    local = a.asname or a.name
                    mod.aliases.setdefault(local, set()).add(f"{source}.{a.name}")
                else:
                    yield source, a.name
    quoted = (n for a in annotations(mod.tree) for n in walk(a))
    for node in itertools.chain(ast.walk(mod.tree), quoted):
        if not isinstance(node, ast.Attribute):
            continue
        chain = []
        value: ast.expr = node
        while isinstance(value, ast.Attribute):
            chain.append(value.attr)
            value = value.value
        if not isinstance(value, ast.Name) or value.id not in mod.aliases:
            continue
        for source in mod.aliases[value.id]:
            for part in reversed(chain):
                if f"{source}.{part}" not in known:
                    yield source, part
                    break
                source = f"{source}.{part}"


def entry_points(
    repo: Path, *, packages: tuple[Package, ...]
) -> Iterator[tuple[str, str]]:
    for pkg in packages:
        with (repo / pkg.root / "pyproject.toml").open("rb") as f:
            scripts = tomllib.load(f).get("project", {}).get("scripts", {})
        for target in scripts.values():
            source, _, name = target.partition(":")
            yield source, name.split(".")[0]


def type_param_bounds(
    node: ast.FunctionDef | ast.AsyncFunctionDef | ast.ClassDef | ast.TypeAlias,
) -> list[ast.expr | None]:
    """Returns the bounds and defaults of the generic type parameters of node."""
    exprs: list[ast.expr | None] = []
    for tp in node.type_params:
        if isinstance(tp, ast.TypeVar):
            exprs.append(tp.bound)
        # Python 3.13 added type parameter defaults.
        exprs.append(getattr(tp, "default_value", None))
    return exprs


def signature(fn: ast.FunctionDef | ast.AsyncFunctionDef) -> list[ast.expr | None]:
    args = fn.args
    params = [*args.posonlyargs, *args.args, *args.kwonlyargs]
    params += [a for a in (args.vararg, args.kwarg) if a is not None]
    return [a.annotation for a in params] + [fn.returns, *type_param_bounds(fn)]


def annotations(tree: ast.Module) -> Iterator[ast.expr]:
    """Yields every type expression in tree, nested scopes included: annotations,
    type alias values, and class bases."""
    for n in ast.walk(tree):
        if isinstance(n, ast.FunctionDef | ast.AsyncFunctionDef):
            yield from (e for e in signature(n) if e is not None)
        elif isinstance(n, ast.AnnAssign):
            yield n.annotation
            if n.value is not None and "TypeAlias" in ast.unparse(n.annotation):
                yield n.value
        elif isinstance(n, ast.TypeAlias):
            yield n.value
        elif isinstance(n, ast.ClassDef):
            yield from n.bases


def annotation_names(node: ast.stmt) -> set[str]:
    """Returns the names that the signature, bases, or annotations of node read."""
    exprs: list[ast.expr | None] = []
    if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
        exprs += signature(node)
    elif isinstance(node, ast.ClassDef):
        exprs += node.bases + type_param_bounds(node)
        for item in node.body:
            if isinstance(item, ast.AnnAssign):
                exprs.append(item.annotation)
            elif isinstance(item, ast.FunctionDef | ast.AsyncFunctionDef) and (
                not item.name.startswith("_")
                or (item.name.startswith("__") and item.name.endswith("__"))
            ):
                exprs += signature(item)
    elif isinstance(node, ast.AnnAssign):
        exprs.append(node.annotation)
        if node.value is not None and "TypeAlias" in ast.unparse(node.annotation):
            exprs.append(node.value)
    elif isinstance(node, ast.TypeAlias):
        exprs += [node.value, *type_param_bounds(node)]
    elif isinstance(node, ast.Assign):
        exprs.append(node.value)
    names: set[str] = set()
    for e in exprs:
        if e is not None:
            names |= expression_names(e)
    return names


def expression_names(expr: ast.expr) -> set[str]:
    """Returns the names that expr reads, including those in quoted annotations."""
    return {n.id for n in walk(expr) if isinstance(n, ast.Name)}


def walk(tree: ast.AST) -> Iterator[ast.AST]:
    """Walks tree like ast.walk, and also walks each string that parses as an
    expression, such as a quoted annotation."""
    for n in ast.walk(tree):
        yield n
        if isinstance(n, ast.Constant) and isinstance(n.value, str):
            try:
                quoted = ast.parse(n.value, mode="eval")
            except SyntaxError:
                # A string that is not an expression is not an annotation.
                continue
            yield from walk(quoted.body)


def check(repo: Path, *, packages: tuple[Package, ...] = PACKAGES) -> list[Finding]:
    """Returns the wrongly public names in packages, sorted by path and line."""
    modules = load(repo, packages=packages)
    known = {m.name for m in modules if not m.test or m.utility}
    defs: dict[tuple[str, str], tuple[Module, ast.stmt]] = {}
    for m in modules:
        if not m.checked or m.generated or m.path.name == "__init__.py":
            continue
        for node in m.tree.body:
            for name in top_level_names(node):
                if not name.startswith("_"):
                    defs[(m.name, name)] = (m, node)
    used: set[tuple[str, str]] = set(entry_points(repo, packages=packages))
    tested: set[tuple[str, str]] = set()
    for m in modules:
        for source, name in imports(m, known=known):
            if source == m.name:
                continue
            keys = (
                [k for k in defs if k[0] == source] if name == "*" else [(source, name)]
            )
            for key in keys:
                owner = defs.get(key)
                if m.test and owner is not None and not owner[0].utility:
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
