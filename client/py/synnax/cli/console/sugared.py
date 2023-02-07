#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, TypeVar, overload

from pydantic import BaseModel
from synnax.cli.console.protocol import Print, Prompt, R

V = TypeVar("V", BaseModel, float, str, int)

class SugaredConsole:
    enabled: bool
    prompt: Prompt
    print: Print 

    def __init__(self, prompt: Prompt, print: Print, enabled: bool = True):
        self.prompt = prompt
        self.print = print
        self.enabled = enabled

    def info(self, message: str) -> None:
        self.print.info(message)

    def success(self, message: str) -> None:
        self.print.success(message)

    def warn(self, message: str) -> None:
        self.print.warn(message)

    def error(self, message: str) -> None:
        self.print.error(message)

    def table(self, columns: list[str], rows: list[dict]) -> None:
        self.print.table(columns, rows)

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] = str,
        choices: list[str] | None = None,
        password: bool = False,
        *,
        arg: R | None = None,
        arg_name: R | None = None,
        default: R = ...,
    ) -> R:
        ...

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] = str,
        choices: list[R] | None = None,
        *,
        arg: R | None = None,
        arg_name: R | None = None,
        default: None = None,
    ) -> R | None:
        ...

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] = str,
        choices: list[R] | None = None,
        *,
        arg: R | None = None,
        arg_name: R | None = None,
    ) -> R:
        ... 

    def ask(
        self,
        question: str,
        type_: type[R] = str,
        choices: list[R] | None = None,
        *_,
        **kwargs,
    ) -> R | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask(
            question=question, 
            type_=type_,
            choices=choices,
            default=default
        )
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("You must provide a value.")
        return self.ask(question, type_, choices, **kwargs)


    def _validate(self, kwargs: dict[str, Any]) -> tuple[R | None, R | None, bool, bool]:
        has_default = "default" in kwargs

        default = kwargs.get("default", None)
        arg = kwargs.get("arg", None)
        if arg is not None:
            return arg, default, True, has_default

        arg_name = kwargs.get("arg_name", None)
        if not self.enabled and not has_default:
            raise ValueError(f"Missing required argument: {arg_name}")

        return default, default, False, has_default
    
    @overload
    def select(
        self,
        rows: list[R],
        columns: list[str],
        *,
        arg: R | None = None,
        arg_name: str | None = None,
        default: R = ...,
    ) -> tuple[R, int]:
        ...

    @overload
    def select(
        self,
        rows: list[dict[str, Any]],
        columns: list[str],
        key: str,
        *,
        arg: R | None = None,
        arg_name: str | None = None,
        default: R = ...,
    ) -> tuple[R, int]:
        ...

    @overload
    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        columns: list[str] | None = None,
        *,
        arg: R | None = None,
        arg_name: str | None = None,
        default: None = None,
    ) -> tuple[R, int] | None:
        ...

    @overload
    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        columns: list[str] | None = None,
        *,
        arg: R | None = None,
        arg_name: str | None = None,
    ) -> tuple[R, int]:
        ...

    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        columns: list[str] | None = None,
        key: str | None = None,
        *_,
        **kwargs
    ) -> V | None:
        """Prompts the user to select a row from a table.

        :param ctx: The current flow Context.
        :param columns: The columns of the table.
        :param rows: The rows of the table.
        :param allow_none: Whether to allow the user to select nothing.
        :param default: The default option to select. If a default is provided,
        allow_none is ignored.
        :returns: The index of the selected row.py or None if nothing was selected.
        """
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        _rows = list()
        if columns is None:
            columns = list()
            for row in rows:
                if isinstance(row, BaseModel):
                    for k, v in row.dict().items():
                        if k not in columns:
                            columns.append(k)
                else:
                    columns.append("option")

        default_idx = default
        for row in rows:
            if row == default:
                default_idx = len(_rows) 
            if isinstance(row, BaseModel):
                _rows.append(row.dict())
            else:
                _rows.append({"value": row})

        self.table(columns=["option", *columns],rows=_rows)
        i = self.prompt.ask(
            "Select an option #", 
            int, 
            choices=[i for i in range(len(rows))], 
            default=default_idx,
        )

        if i is not None:
            return rows[i]
        if has_default:
            return default
        if self.print is not None:
            self.print.error("You must make a selection.")
        return self.select(rows, columns, **kwargs)
