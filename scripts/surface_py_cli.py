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
walks the real command tree, so it runs in the environment of the client.

    uv run --project client/py python scripts/surface_py_cli.py
"""

import json
import sys
from importlib.metadata import distribution
from typing import Any

import click

HELP_FIELDS = {"help", "short_help", "epilog"}


def walk(info: dict[str, Any], *, path: str, out: list[str]) -> None:
    flags = "".join(f" {k}" for k in ("hidden", "deprecated") if info[k])
    out.append(f"{path} command{flags}")
    for param in info["params"]:
        fields = {k: v for k, v in param.items() if k not in HELP_FIELDS}
        out.append(f"{path} {param['name']} {json.dumps(fields, sort_keys=True)}")
    for name, sub in info.get("commands", {}).items():
        walk(sub, path=f"{path} {name}", out=out)


def report() -> str:
    out: list[str] = []
    for script in distribution("synnax").entry_points.select(group="console_scripts"):
        command = script.load()
        walk(
            command.to_info_dict(click.Context(command, info_name=script.name)),
            path=script.name,
            out=out,
        )
    return "".join(f"{line}\n" for line in sorted(out))


def main() -> None:
    sys.stdout.write(report())


if __name__ == "__main__":
    main()
