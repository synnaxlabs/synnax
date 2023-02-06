#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel
from synnax.cli.flow import Context
from typing import TypeVar
 
V = TypeVar("V", BaseModel, str, int, float)


def select_from_table(
    ctx: Context,
    columns: list[str],
    rows: list[V],
    default: int | None = None,
    *,
    arg: V | None = None,
    arg_name: str | None = None,
) -> int | None:
    """Prompts the user to select a row from a table.

    :param ctx: The current flow Context.
    :param columns: The columns of the table.
    :param rows: The rows of the table.
    :param allow_none: Whether to allow the user to select nothing.
    :param default: The default option to select. If a default is provided,
    allow_none is ignored.
    :returns: The index of the selected row.py or None if nothing was selected.
    """
    _rows = list()
    for row in rows:
        if isinstance(row, BaseModel):
            _rows.append(row.dict())
        else:
            _rows.append(BaseModel(**{columns[0]: row}))
    ctx.console.table(columns=["option", *columns],rows=_rows)
    i = ctx.console.ask_int("Select an option #", bound=(0, len(rows)))
    return None if i == "None" else i


def select_simple(
    ctx: Context,
    choices: list[str],
    default: int | None = None,
    required: bool = True,
) -> int | None:
    """Prompts the user to select a choice from the given choices.

    :param ctx: The current flow Context.
    :param choices: The choices to select from.
    :param default: The default option to select. If a default is provided,
    allow_none is ignored.
    :returns: The index of the selected choice or None if nothing was selected.
    """
    return select_from_table(
        ctx,
        ["choice"],
        [{"choice": choice} for choice in choices],
        default,
        required,
    )
