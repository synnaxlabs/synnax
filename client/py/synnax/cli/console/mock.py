#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, TextIO, Type

from pydantic import BaseModel

from synnax.cli.console.protocol import (
    Console,
    Print,
    Prompt,
    R,
    assign_default_ask_type,
)
from synnax.cli.console.rich import RichConsole


class Entry(BaseModel, Generic[R]):
    message: str | None = None
    columns: list[str] | None = None
    rows: list[dict] | None = None
    choices: list[R] | None = None
    default: R | None = None
    response: R | None = None
    type_: Type[R] | None = None
    password: bool | None = None


class Output(BaseModel):
    entries: list[Entry]

    def __init__(self, entries: list[Entry] | None = None):
        super().__init__(entries=entries or list())

    def append(self, entry: Entry):
        assert self.entries is not None
        self.entries.append(entry)

    def write(self, f: TextIO):
        f.write(self.json())


class MockPrint:
    """A mock implementation of the Print protocol for testing purposes."""

    output: Output
    verbose: Console | None

    def __init__(self, output: Output, verbose: bool = False):
        """
        :param output: The output list to append entries to.
        """
        self.output = output
        self.verbose = None if not verbose else RichConsole()

    def _(self) -> Print:
        return self

    def info(self, message: str):
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.info(message)

    def error(self, message: str):
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.error(message)

    def warn(self, message: str):
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.warn(message)

    def success(self, message: str):
        self.output.append(Entry(message=message))
        if self.verbose is not None:
            self.verbose.success(message)

    def table(self, columns: list, rows: list):
        self.output.append(Entry(columns=columns, rows=rows))
        if self.verbose is not None:
            self.verbose.table(columns, rows)


class MockPrompt:
    """A mock implementation of the Prompt protocol for testing purposes."""

    output: Output
    responses: list

    def __init__(self, output: Output, responses: list):
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
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        default: R | None = None,
        password: bool = False,
    ) -> R | None:
        e = Entry(
            message=question,
            choices=choices,
            default=default,
            type_=assign_default_ask_type(type_, choices, default),
            password=password,
        )
        e.response = self.responses.pop(0) if len(self.responses) > 0 else default
        if type(e.response) != e.type_:
            raise TypeError(
                f"""
                Mock Prompt: Invalid response type
                Question: {question}
                Expected type: {type_}
                Actual response: {e.response}
                """
            )
        return e.response


class MockConsole(MockPrint, MockPrompt):
    """A mock implementation of the Console protocol for testing purposes."""

    def __init__(
        self,
        output: Output = Output(),
        responses: list | None = None,
        verbose: bool = False,
    ):
        """
        :param output: The output list to append entries to.
        :param responses: A list of responses to return in order. These responses
        must be valid for the type of prompt being used.
        """
        MockPrint.__init__(self, output, verbose)
        MockPrompt.__init__(self, output, responses or list())
