#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import sys
from collections.abc import Callable, Sequence

from synnax.cli.login import login

_COMMANDS: dict[str, Callable[[], bool]] = {"login": login}


def parser() -> argparse.ArgumentParser:
    """Returns the parser for the sy command."""
    root = argparse.ArgumentParser(prog="sy", description="The Synnax command line.")
    commands = root.add_subparsers(dest="command", required=True)
    commands.add_parser(
        "login",
        help="Log in to a Core and save an API key for scripts on this computer.",
        description=(
            "Prompts for the address and credentials of a Core, creates an API key, "
            "and saves the key and the address."
        ),
    )
    return root


def synnax(argv: Sequence[str] | None = None) -> None:
    """Runs the sy command. Exits with a non-zero code when the command fails."""
    args = parser().parse_args(argv)
    try:
        ok = _COMMANDS[args.command]()
    except KeyboardInterrupt:
        print(file=sys.stderr)
        sys.exit(130)
    except EOFError:
        print("\nNo input. Cancelled.", file=sys.stderr)
        sys.exit(1)
    if not ok:
        sys.exit(1)
