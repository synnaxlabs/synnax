#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Protocol, TypeVar

R = TypeVar("R", str, int, float, bool, None)


class Prompt(Protocol):
    """A protocol class for an entity that can prompt the user for input."""

    def ask(
        self,
        question: str,
        type_: type[R] | None = None,
        choices: list[R] | None = None,
        default: R | None = None,
        password: bool = False,
    ) -> R | None:
        """Asks the user a question and returns their response.

        :param question: The question to ask the user.
        :param choices: A list of choices the user can select from. If provided,
        the user will be prompted to select one of the choices. A response not
        in the list will be rejected.
        :param default: A default value to use if the user does not provide a
        response. If provided, the user can press enter to select the default
        value.
        :return: The user's response.
        """
        ...


class Print(Protocol):
    """A protocol class for an entity that can print messages to the console."""

    def info(
        self,
        message: str,
    ) -> None:
        """Prints an informational message to the console.

        :param message: The message to print.
        """
        ...

    def warn(
        self,
        message: str,
    ) -> None:
        """Prints a warning message to the console.

        :param message: The message to print.
        """
        ...

    def error(
        self,
        message: str,
    ) -> None:
        """Prints an error message to the console.

        :param message: The message to print.
        """
        ...

    def success(
        self,
        message: str,
    ) -> None:
        """Prints a success message to the console.

        :param message: The message to print.
        """
        ...

    def table(
        self,
        columns: list[str],
        rows: list[dict],
    ):
        """Prints a table to the console.

        :param columns: A list of column names.
        :param rows: A list of dictionaries, where each dictionary represents a row.py. The
        keys of the dictionary should match the column names.
        """
        ...


class Console(Prompt, Print, Protocol):
    """A protocol class for an entity that can print messages to the console and prompt
    the user for input.
    """

    ...


def assign_default_ask_type(
    type_: type[R] | None, choices: list[R] | None, default: R | None
) -> type[R]:
    """Assigns a default type to the ask function.

    :param type_: The type to assign.
    :param default: The default value.
    :param choices: The list of choices.
    :return: The type.
    """
    if type_ is None:
        if choices is not None:
            type_ = type(choices[0])
        elif default is not None:
            type_ = type(default)  # type: ignore
        else:
            type_ = str  # type: ignore
    return type_  # type: ignore
