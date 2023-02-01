#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from synnax.telem import DataType, TimeSpan
from synnax.cli.channel import select_from_table
from synnax.cli.flow import Context


def prompt_data_type_select(ctx: Context, required: bool = True) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    i = select_from_table(
        ctx,
        ["data_type"],
        [{"data_type": name.string()} for name in DataType.ALL],
        required,
    )
    return DataType.ALL[i] if i is not None else None


def ask_time_units_select(
    ctx: Context, required: bool = True, question: str | None = None
) -> str | None:
    """Prompts the user to select a time unit from a list of all available time
    units.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    if question is not None:
        ctx.console.ask(question)
    opts = list(TimeSpan.UNITS.keys())
    i = select_from_table(
        ctx,
        ["unit"],
        [{"unit": unit} for unit in opts],
        required,
    )
    return opts[i] if i is not None else None
