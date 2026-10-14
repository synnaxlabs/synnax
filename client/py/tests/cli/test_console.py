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

from synnax.cli.console.mock import MockPrint, Output
from synnax.cli.console.sugared import SugaredConsole


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
class TestSugaredConsole:
    def test_ask_repeats_until_answered(self) -> None:
        output = Output()
        prompt = _ScriptedPrompt([None, None, "secret"])
        console = SugaredConsole(prompt=prompt, print=MockPrint(output))
        assert console.ask("Password", str, password=True) == "secret"
        assert prompt.passwords == [True, True, True]
        assert len(output.entries) == 2

    def test_ask_returns_none_with_none_default(self) -> None:
        prompt = _ScriptedPrompt([None])
        console = SugaredConsole(prompt=prompt, print=MockPrint(Output()))
        assert console.ask("Name", str, default=None) is None
        assert len(prompt.passwords) == 1
