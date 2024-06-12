#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Generic, NotRequired, TypedDict, Unpack, overload

from synnax.cli.console.protocol import Print, Prompt, R
from synnax.exceptions import ValidationError


class AskKwargs(TypedDict, Generic[R]):
    arg: NotRequired[R | None]
    arg_name: NotRequired[str]


class NoneDefaultAskKwargs(AskKwargs[R]):
    default: NotRequired[R | None]


class DefaultAskKwargs(AskKwargs[R]):
    default: R


class SugaredConsole:
    enabled: bool
    prompt: Prompt
    print: Print

    def __init__(self, prompt: Prompt, print: Print, enabled: bool = True):
        self.prompt = prompt
        self.print = print
        self.enabled = enabled

    def info(self, message: str) -> None:
        if self.enabled:
            self.print.info(message)

    def success(self, message: str) -> None:
        if self.enabled:
            self.print.success(message)

    def warn(self, message: str) -> None:
        if self.enabled:
            self.print.warn(message)

    def error(self, message: str) -> None:
        if self.enabled:
            self.print.error(message)

    def table(self, columns: list[str], rows: list[dict]) -> None:
        if self.enabled:
            self.print.table(columns, rows)

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        password: bool = False,
        **kwargs: Unpack[AskKwargs[R]],
    ) -> R:
        ...

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] | None = None,
        choices: list[str] | None = None,
        password: bool = False,
        **kwargs: Unpack[DefaultAskKwargs[R]],
    ) -> R:
        ...

    @overload
    def ask(
        self,
        question: str,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        password: bool = False,
        **kwargs: Unpack[NoneDefaultAskKwargs[R]],
    ) -> R | None:
        ...

        ...

    def ask(
        self,
        question: str,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        password: bool = False,
        **kwargs: Unpack[NoneDefaultAskKwargs[R]],
    ) -> R | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask(
            question=question,
            type_=type_,
            choices=choices,
            default=default,
            password=password,
        )
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("You must provide a value.")
        return self.ask(question, type_, choices, **kwargs)

    def _validate(
        self, kwargs: NoneDefaultAskKwargs[R]
    ) -> tuple[R | None, R | None, bool, bool]:
        has_default = "default" in kwargs

        default = kwargs.get("default", None)
        arg = kwargs.get("arg", None)
        if arg is not None:
            return arg, default, True, has_default

        arg_name = kwargs.get("arg_name", None)
        if not self.enabled and not has_default:
            raise ValueError(f"Missing required argument: {arg_name}")

        return default, default, not self.enabled, has_default

    @overload
    def select(
        self, rows: list[R], type_: type[R] = str, **kwargs: Unpack[DefaultAskKwargs[R]]
    ) -> tuple[R, int]:
        ...

    @overload
    def select(
        self,
        rows: list[dict[str, Any]],
        type_: type[R] = str,
        columns: list[str] | None = None,
        key: str | None = None,
        **kwargs: Unpack[DefaultAskKwargs[R]],
    ) -> tuple[R, int]:
        ...

    @overload
    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        type_: type[R],
        columns: list[str] | None = None,
        **kwargs: Unpack[AskKwargs[R]],
    ) -> tuple[R, int]:
        ...

    @overload
    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        type_: type[R],
        columns: list[str] | None = None,
        **kwargs: Unpack[NoneDefaultAskKwargs[R]],
    ) -> tuple[R | None, int | None]:
        ...

    def select(
        self,
        rows: list[R] | list[dict[str, Any]],
        type_: type[R] = str,
        columns: list[str] | None = None,
        key: str | None = None,
        **kwargs: Unpack[NoneDefaultAskKwargs[R]],
    ) -> tuple[R | None, int | None]:
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
        if len(rows) == 0:
            raise ValidationError("No rows to select from.")
        if isinstance(rows[0], dict) and key is None:
            raise ValidationError("Missing key argument.")
        _key: str = key or "value"

        _rows = list()
        default_idx = 0
        no_cols = columns is None
        _columns = columns or list()

        for i, row in enumerate(rows):
            if isinstance(row, dict):
                is_default = row[_key] == default
                _rows.append(row)
                if no_cols:
                    for k in row.keys():
                        if k not in _columns:
                            _columns.append(k)
            else:
                is_default = row == default
                key = "value"
                if len(_columns) > 0:
                    key = _columns[0]
                _rows.append({"choice": str(i), key: row})
            if is_default:
                default_idx = len(_rows) - 1

        if len(_columns) == 0:
            _columns = ["value"]

        if "choice" not in _columns:
            _columns = ["choice"] + _columns

        if should_return:
            return v, default_idx

        self.table(columns=_columns, rows=_rows)
        i = self.ask(
            "Select an option #",
            int,
            choices=[i for i in range(len(rows))],
            default=default_idx,
        )

        if i is not None:
            r = rows[i]
            assert r is not None
            return (r[_key], i) if isinstance(r, dict) else (r, i)
        if has_default:
            return default, default_idx
        if self.print is not None:
            self.print.error("You must make a selection.")
        return self.select(type_, rows, columns, key, **kwargs)  # type: ignore
