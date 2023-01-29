#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from synnax.io import IOFactory, RowReader
from synnax.cli.flow import Context

io_factory = IOFactory()


def prompt_file(ctx: Context) -> Path | None:
    """Prompts the user for a file path."""
    _fp = ctx.console.ask("File path")
    assert _fp is not None
    fp = Path(_fp)
    if not fp.exists():
        ctx.console.error(f"File does not exist: {fp}")
        return prompt_file(ctx)
    if not fp.is_file():
        ctx.console.error(f"Not a file: {fp}")
        return prompt_file(ctx)
    ctx.console.success(f"File found: {fp}")
    return fp


def prompt_new_reader(ctx: Context) -> RowReader | None:
    """Prompts the user for a file path and returns a new reader for that file."""
    fp = prompt_file(ctx)
    if fp is None:
        return None
    return io_factory.new_reader(fp)
