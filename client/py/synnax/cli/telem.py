#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import Unpack

from synnax.cli.console.sugared import AskKwargs
from synnax.cli.flow import Context
from synnax.telem import DataType, TimeSpan, TimeSpanUnits


def select_data_type(
    ctx: Context,
    **kwargs: Unpack[AskKwargs[str]],
) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    return DataType(
        ctx.console.select(
            rows=[str(name) for name in DataType.ALL],
            type_=str,
            columns=["data_type"],
            **kwargs,
        )[0]
    )


def ask_time_units_select(
    ctx: Context,
    question: str | None = None,
    **kwargs: Unpack[AskKwargs[str]],
) -> TimeSpanUnits:
    """Prompts the user to select a time unit from a list of all available time
    units.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    if question is not None:
        ctx.console.info(question)
    return ctx.console.select(
        rows=["iso", *list(TimeSpan.UNITS.keys())],
        type_=str,
        columns=["unit"],
        **kwargs,
    )[0]
