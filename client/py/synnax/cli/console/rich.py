#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import cast

from rich import print
from rich.prompt import Confirm, FloatPrompt, IntPrompt, Prompt

from synnax.cli.console.protocol import Console, R

_SYNNAX_PRIMARY_Z = "#3774D0"


class Rich:
    """A Console implementation using Rich."""

    info_color: str
    warn_color: str
    error_color: str
    success_color: str

    def __init__(
        self,
        *,
        info_color: str = _SYNNAX_PRIMARY_Z,
        warn_color: str = "yellow",
        error_color: str = "red",
        success_color: str = "green",
    ):
        self.info_color = info_color
        self.warn_color = warn_color
        self.error_color = error_color
        self.success_color = success_color

    def _(self) -> Console:
        return self

    def info(self, message: str) -> None:
        print(f"[{self.info_color}]{message}[/]")
        return

    def warn(self, message: str) -> None:
        print(f"[{self.warn_color}]{message}[/]")
        return

    def error(self, message: str) -> None:
        print(f"[{self.error_color}]{message}[/]")
        return

    def success(self, message: str) -> None:
        print(f"[{self.success_color}]{message}[/]")
        return

    def table(
        self,
        columns: list[str],
        rows: list[dict[str, str]],
    ) -> None:
        from rich.table import Table

        table = Table(show_header=True, header_style="bold magenta")
        for column in columns:
            table.add_column(column)
        for row in rows:
            # order the row.py by the columns
            table.add_row(*[row[column] for column in columns])

        print(table)
        return

    def ask(
        self,
        question: str,
        *,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        default: R | None = None,
        password: bool = False,
    ) -> R | None:
        resolved: type[object] = str
        if type_ is not None:
            resolved = type_
        elif default is not None:
            resolved = type(default)
        elif choices is not None and len(choices) > 0:
            resolved = type(choices[0])
        # mypy checks this body once per constraint of R, so each branch casts its
        # prompt's result back to R.
        if resolved is bool:
            return cast(
                R | None,
                Confirm.ask(
                    question,
                    default=default,
                    show_default=True,
                    show_choices=True,
                ),
            )
        if resolved is int:
            return cast(
                R | None,
                IntPrompt.ask(
                    question,
                    default=default,
                    choices=[str(choice) for choice in choices] if choices else None,
                ),
            )
        if resolved is float:
            return cast(R | None, FloatPrompt.ask(question, default=default))
        return cast(
            R | None,
            Prompt.ask(
                question,
                choices=[str(choice) for choice in choices] if choices else None,
                default=default,
                password=password,
            ),
        )
