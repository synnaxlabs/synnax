#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from synnax.telem import DataType, TimeSpan
from synnax.cli.flow import Context


def select_data_type(
    ctx: Context,
    *,
    arg: str | None = None,
    arg_name: str | None = None,
) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    return DataType(
        ctx.console.select(
            ["data_type"],
            [str(name) for name in DataType.ALL],
            arg=arg,
            arg_name=arg_name,
        )
    )


def ask_time_units_select(
    ctx: Context,
    question: str | None = None,
    *,
    arg: str | None = None,
    arg_name: str | None = None,
) -> str:
    """Prompts the user to select a time unit from a list of all available time
    units.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    if question is not None:
        ctx.console.info(question)
    opts = list(TimeSpan.UNITS.keys())
    return select_from_table(ctx, ["unit"], opts, arg=arg, arg_name=arg_name)
