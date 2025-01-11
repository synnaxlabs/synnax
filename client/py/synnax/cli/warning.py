#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from click import Context, echo, style

correct_name = "synnaxkit"


def warning(ctx: Context) -> None:
    """Warns the user if a deprecated command is used."""
    name = ctx.find_root().info_name
    if name != correct_name:
        styled_error = style("DEPRECATION WARNING:", fg="red")
        styled_name = style(name, fg="cyan")
        styled_correct_name = style(correct_name, fg="cyan")
        echo(
            f"{styled_error} The {styled_name} command has been deprecated and will be removed in a future release. Please use {styled_correct_name} instead."
        )
