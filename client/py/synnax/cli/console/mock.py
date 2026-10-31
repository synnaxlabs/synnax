#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Generic, TextIO

from pydantic import BaseModel

from synnax.cli.console.protocol import (
    Console,
    Print,
    Prompt,
    R,
    assign_default_ask_type,
)
from synnax.cli.console.rich import Rich


class Entry(BaseModel, Generic[R]):
    message: str | None = None
    columns: list[str] | None = None
    rows: list[dict[str, str]] | None = None
    choices: list[R] | None = None
    default: R | None = None
    response: R | None = None
    type_: type[str | int | float | bool] | None = None
    password: bool | None = None


class Output(BaseModel):
    entries: list[Entry[Any]]

    def __init__(self, entries: list[Entry[Any]] | None = None):
        super().__init__(entries=entries or [])

    def append(self, entry: Entry[Any]) -> None:
        assert self.entries is not None
        self.entries.append(entry)

    def write(self, f: TextIO) -> None:
        f.write(self.model_dump_json())


class MockPrint:
    """A mock implementation of the Print protocol for testing purposes."""

    output: Output
    verbose: Console | None

    def __init__(self, *, output: Output, verbose: bool = False):
        """
        :param output: The output list to append entries to.
        """
        self.output = output
        self.verbose = None if not verbose else Rich()

    def _(self) -> Print:
        return self

    def info(self, message: str) -> None:
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.info(message)

    def error(self, message: str) -> None:
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.error(message)

    def warn(self, message: str) -> None:
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.warn(message)

    def success(self, message: str) -> None:
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.success(message)

    def table(self, columns: list[str], rows: list[dict[str, str]]) -> None:
        self.output.append(Entry(columns=columns, rows=rows))
        if self.verbose is not None:
            self.verbose.table(columns, rows)


class MockPrompt:
    """A mock implementation of the Prompt protocol for testing purposes."""

    output: Output
    responses: list[Any]

    def __init__(self, *, output: Output, responses: list[Any]):
        """
        :param output: The output list to append entries to.
        :param responses: A list of responses to return in order. These responses
        must be valid for the type of prompt being used.
        """
        self.output = output
        self.responses = responses

    def _(self) -> Prompt:
        return self

    def ask(
        self,
        question: str,
        *,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        default: R | None = None,
        password: bool = False,
    ) -> R | None:
        resolved_type = assign_default_ask_type(
            type_=type_, choices=choices, default=default
        )
        response: R | None = (
            self.responses.pop(0) if len(self.responses) > 0 else default
        )
        Entry(
            message=question,
            choices=choices,
            default=default,
            type_=resolved_type,
            password=password,
            response=response,
        )
        if type(response) is not resolved_type:
            raise TypeError(f"""
                Mock Prompt: Invalid response type
                Question: {question}
                Expected type: {type_}
                Actual response: {response}
                """)
        return response


class Mock(MockPrint, MockPrompt):
    """A mock implementation of the Console protocol for testing purposes."""

    def __init__(
        self,
        *,
        output: Output | None = None,
        responses: list[Any] | None = None,
        verbose: bool = False,
    ):
        """
        :param output: The output list to append entries to.
        :param responses: A list of responses to return in order. These responses
        must be valid for the type of prompt being used.
        """
        output = output if output is not None else Output()
        MockPrint.__init__(self, output=output, verbose=verbose)
        MockPrompt.__init__(self, output=output, responses=responses or [])

    def _(self) -> Console:
        return self
