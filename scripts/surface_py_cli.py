#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Prints the commands, arguments, and options of the Python client CLI, one sorted line
each, without help text.

Compare the output of two revisions to see if a pull request changes the CLI. The script
walks the real parser that the module of each console script returns from `parser()`.

    uv run --project scripts scripts/surface_py_cli.py
"""

import argparse
import importlib
import json
import sys
from importlib.metadata import distribution
from typing import Any


def _fields(action: argparse.Action) -> dict[str, Any]:
    return {
        "choices": None if action.choices is None else sorted(map(str, action.choices)),
        "default": repr(action.default),
        "nargs": action.nargs,
        "option_strings": sorted(action.option_strings),
        "required": action.required,
        "type": getattr(action.type, "__name__", None),
    }


def walk(parser: argparse.ArgumentParser, *, path: str, out: list[str]) -> None:
    out.append(f"{path} command")
    position = 0
    for action in parser._actions:
        if isinstance(action, argparse._HelpAction):
            continue
        if isinstance(action, argparse._SubParsersAction):
            required = json.dumps({"required": action.required})
            out.append(f"{path} {action.dest} {required}")
            for name, sub in action.choices.items():
                walk(sub, path=f"{path} {name}", out=out)
            continue
        fields = _fields(action)
        if not action.option_strings:
            fields["position"] = position
            position += 1
        out.append(f"{path} {action.dest} {json.dumps(fields, sort_keys=True)}")


def report() -> str:
    out: list[str] = []
    for script in distribution("synnax").entry_points.select(group="console_scripts"):
        module = importlib.import_module(script.module)
        walk(module.parser(), path=script.name, out=out)
    return "".join(f"{line}\n" for line in sorted(out))


def main() -> None:
    sys.stdout.write(report())


if __name__ == "__main__":
    main()
