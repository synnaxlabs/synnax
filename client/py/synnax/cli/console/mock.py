#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Generic, TextIO, Type
from pydantic import BaseModel

from synnax.cli.console.protocol import Print, Prompt, R


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

    def __init__(self, output: Output):
        """
        :param output: The output list to append entries to.
        """
        self.output = output

    def _(self) -> Print:
        return self

    def info(self, message: str):
        self.output.append(Entry(message=message))

    def error(self, message: str):
        self.output.append(Entry(message=message))

    def warn(self, message: str):
        self.output.append(Entry(message=message))

    def success(self, message: str):
        self.output.append(Entry(message=message))

    def table(self, columns: list, rows: list):
        self.output.append(Entry(columns=columns, rows=rows))


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
        type_: Type[R] = str,
        choices: list[R] | None = None,
        default: R | None = None,
        password: bool = False,
    ):
        e = Entry[R](
            message=question,
            choices=choices,
            default=default,
            type_=type_,  # type: ignore
            password=password,
        )
        e.response = self.responses.pop(0) or default
        if type(e.response) != type_:
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

    def __init__(self, output: Output = Output(), responses: list | None = None):
        """
        :param output: The output list to append entries to.
        :param responses: A list of responses to return in order. These responses
        must be valid for the type of prompt being used.
        """
        self.output = output
        self.responses = responses or list()
