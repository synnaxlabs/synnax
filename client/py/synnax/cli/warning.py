#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from click import Context, echo, style

CORRECT_NAME = "sy"
STYLED_ERROR = style("DEPRECATION WARNING:", fg="red")
STYLED_CORRECT_NAME = style(CORRECT_NAME, fg="cyan")


def warning(ctx: Context) -> None:
    """Warns the user if a deprecated command is used."""
    name = ctx.find_root().info_name
    if name != CORRECT_NAME:
        styled_name = style(name, fg="cyan")
        echo(
            f"{STYLED_ERROR} The {styled_name} command has been deprecated and will be removed in a future release. Please use {STYLED_CORRECT_NAME} instead."
        )
