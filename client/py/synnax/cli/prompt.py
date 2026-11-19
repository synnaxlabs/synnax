#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import getpass
import sys
from collections.abc import Callable
from typing import TextIO

_YES = {"y", "yes"}
_NO = {"n", "no"}


class Prompt:
    """Prompt asks questions on a terminal and prints messages to it.

    :param read: Prints the given question and returns the line the user enters.
    :param read_secret: Like read, but does not echo the line.
    :param out: The stream for messages.
    :param err: The stream for errors.
    """

    def __init__(
        self,
        *,
        read: Callable[[str], str] = input,
        read_secret: Callable[[str], str] = getpass.getpass,
        out: TextIO = sys.stdout,
        err: TextIO = sys.stderr,
    ):
        self._read = read
        self._read_secret = read_secret
        self._out = out
        self._err = err

    def ask(self, question: str, *, default: str = "") -> str:
        """Returns the answer to question, or default when the answer is empty."""
        suffix = f" [{default}]" if default else ""
        return self._read(f"{question}{suffix}: ").strip() or default

    def ask_int(self, question: str, *, default: int) -> int:
        """Asks question until the answer is empty or an integer."""
        while True:
            answer = self.ask(question, default=str(default))
            try:
                return int(answer)
            except ValueError:
                self.error(f"{answer!r} is not an integer.")

    def confirm(self, question: str, *, default: bool = False) -> bool:
        """Asks a yes or no question until the answer is empty, yes, or no."""
        suffix = " [Y/n]: " if default else " [y/N]: "
        while True:
            answer = self._read(f"{question}{suffix}").strip().lower()
            if not answer:
                return default
            if answer in _YES:
                return True
            if answer in _NO:
                return False
            self.error("Answer y or n.")

    def secret(self, question: str) -> str:
        """Returns the answer to question without echoing it."""
        return self._read_secret(f"{question}: ")

    def info(self, message: str) -> None:
        print(message, file=self._out)

    def error(self, message: str) -> None:
        print(message, file=self._err)
