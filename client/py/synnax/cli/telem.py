#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import Any, cast

from synnax.cli.flow import Context
from synnax.telem import DataType, TimeSpan, TimeSpanUnits


def select_data_type(
    ctx: Context,
    **kwargs: Any,
) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    selected, _ = ctx.console.select(
        rows=[str(name) for name in DataType.ALL],
        type_=str,
        columns=["data_type"],
        **kwargs,
    )
    if selected is None:
        return None
    return DataType(selected)


def ask_time_units_select(
    ctx: Context,
    question: str | None = None,
    **kwargs: Any,
) -> TimeSpanUnits:
    """Prompts the user to select a time unit from a list of all available time
    units.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    if question is not None:
        ctx.console.info(question)
    unit_rows: list[str] = ["iso", *list(TimeSpan.UNITS.keys())]
    selected, _ = ctx.console.select(
        rows=unit_rows,
        type_=str,
        columns=["unit"],
        **kwargs,
    )
    assert selected is not None
    return cast(TimeSpanUnits, selected)
