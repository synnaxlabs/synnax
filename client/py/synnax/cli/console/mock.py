#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TextIO
from pydantic import BaseModel

from synnax.cli.console.protocol import Print, Prompt

Response = str | int | float | bool | None


class Entry(BaseModel):
    message: str | None = None
    columns: list[str] | None = None
    rows: list[dict] | None = None
    choices: list[str] | None = None
    default: Response = None
    response: Response = None
    required: bool | None = None


class Output(BaseModel):
    entries: list[Entry]

    def __init__(self, entries: list[Entry] | None):
        self.entries = entries or list()

    def append(self, entry: Entry):
        assert self.entries is not None
        self.entries.append(entry)

    def write(self, f: TextIO):
        f.write(self.json())


class MockPrint(Print):
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


class MockPrompt(Prompt):
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
        choices: list[str] | None = None,
        default: str | None = None,
        required: bool = False,
    ):
        e = Entry(message=question, choices=choices, default=default, required=required)
        e.response = self.responses.pop(0) or default
        if type(e.response) != str:
            raise TypeError("Unexpected response type for ask. It should be a string.")
        return e.response

    def ask_int(
        self, question: str, default: int | None = None, required: bool = False
    ):
        e = Entry(message=question, default=default, required=required)
        e.response = self.responses.pop(0) or default
        if type(e.response) != int:
            raise TypeError(
                "Unexpected response type for ask_int. It should be an integer."
            )
        return default

    def ask_float(
        self, question: str, default: float | None = None, required: bool = False
    ):
        e = Entry(message=question, default=default, required=required)
        e.response = self.responses.pop(0) or default
        if type(e.response) != float:
            raise TypeError(
                "Unexpected response type for ask_float. It should be a float."
            )
        return e.response

    def ask_password(self, message: str, required: bool):
        e = Entry(message=message, required=required)
        e.response = self.responses.pop(0)
        if e.response is None:
            raise ValueError("Password cannot be empty.")
        elif type(e.response) != str:
            raise TypeError(
                "Unexpected response type for ask_password. It should be a string."
            )
        return e.response

    def confirm(self, message: str, default: bool = True):
        e = Entry(message=message, default=default)
        e.response = self.responses.pop(0) or default
        if type(e.response) != bool:
            raise TypeError(
                "Unexpected response type for confirm. It should be a boolean."
            )
        return e.response


class MockConsole(MockPrint, MockPrompt):
    """A mock implementation of the Console protocol for testing purposes."""

    def __init__(self, output: Output, responses: list):
        """
        :param output: The output list to append entries to.
        :param responses: A list of responses to return in order. These responses
        must be valid for the type of prompt being used.
        """
        self.output = output
        self.responses = responses
