#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
from collections.abc import Callable

import pytest

from synnax.cli import console

type Answer = Callable[..., None]


@pytest.fixture
def answer(monkeypatch: pytest.MonkeyPatch) -> Answer:
    """Feeds the given lines to the prompt through standard input."""

    def feed(*lines: str) -> None:
        stdin = io.StringIO("".join(f"{line}\n" for line in lines))
        monkeypatch.setattr("sys.stdin", stdin)

    return feed


@pytest.mark.cli
class TestRichAsk:
    def test_ask_str_by_default(self, answer: Answer) -> None:
        """Should return the raw string when no type, default, or choices are set."""
        answer("hello")
        assert console.Rich().ask("Name?") == "hello"

    def test_ask_explicit_bool(self, answer: Answer) -> None:
        """Should confirm a yes or no question when the type is bool."""
        answer("y")
        assert console.Rich().ask("Continue?", type_=bool) is True

    def test_ask_bool_inferred_from_default(self, answer: Answer) -> None:
        """Should infer a bool question from the default and return it on empty."""
        answer("")
        assert console.Rich().ask("Continue?", default=False) is False

    def test_ask_explicit_int(self, answer: Answer) -> None:
        """Should parse the answer as an int when the type is int."""
        answer("42")
        assert console.Rich().ask("Count?", type_=int) == 42

    def test_ask_int_inferred_from_choices(self, answer: Answer) -> None:
        """Should infer an int question from the choices and reject other values."""
        answer("4", "2")
        assert console.Rich().ask("Pick", choices=[1, 2, 3]) == 2

    def test_ask_float_inferred_from_default(self, answer: Answer) -> None:
        """Should infer a float question from the default."""
        answer("1.5")
        assert console.Rich().ask("Rate?", default=0.5) == 1.5

    def test_ask_str_with_choices(self, answer: Answer) -> None:
        """Should accept only one of the given string choices."""
        answer("d", "b")
        assert console.Rich().ask("Pick", choices=["a", "b", "c"]) == "b"

    def test_ask_str_returns_default_on_empty(self, answer: Answer) -> None:
        """Should return the default when the answer is empty."""
        answer("")
        assert console.Rich().ask("Name?", default="synnax") == "synnax"
