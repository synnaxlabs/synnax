#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from x.params import RequiresNamedParams, require_named_params


class TestRequireNamedParams:
    def test_allows_named_params(self) -> None:
        @require_named_params
        def greet(name: str = "") -> str:
            return f"hello {name}"

        assert greet(name="world") == "hello world"

    def test_raises_on_positional(self) -> None:
        @require_named_params
        def greet() -> str:
            return "hello"

        with pytest.raises(RequiresNamedParams, match="only accepts named"):
            greet("a", "b")  # type: ignore[call-arg]

    def test_with_example_params(self) -> None:
        @require_named_params(example_params=("user_id", "12345"))
        def lookup() -> str:
            return ""

        with pytest.raises(RequiresNamedParams, match="user_id='12345'"):
            lookup("a", "b")  # type: ignore[call-arg]

    def test_non_positional_type_error_propagates(self) -> None:
        @require_named_params
        def bad(*, n: int) -> int:
            return n + "oops"  # type: ignore[operator]

        with pytest.raises(TypeError, match="unsupported operand"):
            bad(n=1)
