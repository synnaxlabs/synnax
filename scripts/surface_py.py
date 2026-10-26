#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the public API of a Python package, one sorted line per object.

Compare the output of two revisions to see if a pull request changes the API. The
package is read statically with griffe, so its dependencies need not be installed.

    uv run --project scripts scripts/surface_py.py client/py synnax
"""

import argparse
import re
import sys

import griffe

KIND_PREFIX = {
    griffe.ParameterKind.var_positional: "*",
    griffe.ParameterKind.var_keyword: "**",
}


def flat(expr: object) -> str:
    return re.sub(r"\s+", " ", str(expr)).strip()


def signature(fn: griffe.Function) -> str:
    params = []
    kw_only_marked = False
    for p in fn.parameters:
        if p.kind == griffe.ParameterKind.var_positional:
            kw_only_marked = True
        if p.kind == griffe.ParameterKind.keyword_only and not kw_only_marked:
            params.append("*")
            kw_only_marked = True
        s = KIND_PREFIX.get(p.kind, "") + p.name
        if p.annotation is not None:
            s += f": {flat(p.annotation)}"
        if p.default is not None:
            s += f" = {flat(p.default)}"
        params.append(s)
        if p.kind == griffe.ParameterKind.positional_only:
            params.append("/")
    out = f"({', '.join(params)})".replace(", /, ", ", ")
    if fn.returns is not None:
        out += f" -> {flat(fn.returns)}"
    return out


def labels(obj: griffe.Object) -> str:
    return f" [{', '.join(sorted(obj.labels))}]" if obj.labels else ""


def walk(obj: griffe.Object, out: list[str]) -> None:
    for name in sorted(obj.members):
        member = obj.members[name]
        if not member.is_public:
            continue
        if isinstance(member, griffe.Alias):
            out.append(f"{member.path} = {member.target_path}")
        elif isinstance(member, griffe.Module):
            out.append(f"{member.path} module")
            walk(member, out)
        elif isinstance(member, griffe.Class):
            bases = ", ".join(flat(b) for b in member.bases)
            out.append(f"{member.path} class({bases}){labels(member)}")
            walk(member, out)
        elif isinstance(member, griffe.Function):
            out.extend(
                f"{member.path} overload{signature(o)}" for o in member.overloads or []
            )
            out.append(f"{member.path}{signature(member)}{labels(member)}")
        elif isinstance(member, griffe.Attribute):
            line = member.path
            if member.annotation is not None:
                line += f": {flat(member.annotation)}"
            if member.value is not None:
                line += f" = {flat(member.value)}"
            out.append(line + labels(member))
        else:
            raise ValueError(f"unknown kind {member.kind} of {member.path}")


def report(search_path: str, package: str) -> str:
    root = griffe.load(package, search_paths=[search_path], resolve_aliases=False)
    if not isinstance(root, griffe.Module):
        raise TypeError(f"{package} is not a module")
    out = [f"{root.path} module"]
    walk(root, out)
    return "\n".join(out) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("search_path", help="directory that holds the package")
    parser.add_argument("package", help="name of the package to import")
    args = parser.parse_args()
    sys.stdout.write(report(args.search_path, args.package))


if __name__ == "__main__":
    main()
