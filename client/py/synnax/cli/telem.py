#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from ..telem import DATA_TYPES, TIME_UNITS, DataType
from .channel import select_from_table
from .flow import Context


def prompt_data_type_select(ctx: Context, allow_none: bool = False) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    i = select_from_table(
        ctx,
        ["data_type"],
        [{"data_type": name.string()} for name in DATA_TYPES],
        allow_none=allow_none,
    )
    return DATA_TYPES[i] if i is not None else None


def prompt_time_units_select(ctx: Context, allow_none: bool = False) -> str | None:
    """Prompts the user to select a time unit from a list of all available time
    units.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    opts = list(TIME_UNITS.keys())
    i = select_from_table(
        ctx,
        ["unit"],
        [{"unit": unit} for unit in opts],
        allow_none=allow_none,
    )
    return opts[i] if i is not None else None
