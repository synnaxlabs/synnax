#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest

from synnax.cli import console
from synnax.cli.console.mock import MockPrint, Output


class _ScriptedPrompt:
    """Answers with each response in turn, including None for an empty answer."""

    def __init__(self, responses: list[Any]) -> None:
        self.responses = responses
        self.passwords: list[bool] = []

    def ask(
        self,
        question: str,
        type_: Any = None,
        choices: Any = None,
        default: Any = None,
        password: bool = False,
    ) -> Any:
        self.passwords.append(password)
        return self.responses.pop(0)


@pytest.mark.cli
class TestSugared:
    """Tests for the prompt flow of Sugared.ask."""

    def test_ask_repeats_until_answered(self) -> None:
        """Should ask again after an empty answer when no default is given."""
        output = Output()
        prompt = _ScriptedPrompt([None, None, "secret"])
        c = console.Sugared(prompt=prompt, print=MockPrint(output=output))
        assert c.ask("Password", type_=str, password=True) == "secret"
        assert prompt.passwords == [True, True, True]
        assert len(output.entries) == 2

    def test_ask_returns_none_with_none_default(self) -> None:
        """Should return None for an empty answer when the default is None."""
        prompt = _ScriptedPrompt([None])
        c = console.Sugared(prompt=prompt, print=MockPrint(output=Output()))
        assert c.ask("Name", type_=str, default=None) is None
        assert len(prompt.passwords) == 1
