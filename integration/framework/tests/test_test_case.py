#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable

import pytest
from pydantic import JsonValue

from framework.test_case import param


class TestParam:
    """Reads a test definition parameter with param."""

    def test_should_return_the_default_for_a_missing_key(self) -> None:
        assert param({}, "rate", 10) == 10

    def test_should_return_a_parameter_of_the_default_type(self) -> None:
        params: dict[str, JsonValue] = {"rate": 100, "mode": "a", "headed": True}
        assert param(params, "rate", 10) == 100
        assert param(params, "mode", "") == "a"
        assert param(params, "headed", False) is True

    def test_should_convert_an_int_for_a_float_default(self) -> None:
        value = param({"wait_time": 2}, "wait_time", 0.0)
        assert value == 2.0
        assert isinstance(value, float)

    @pytest.mark.parametrize(
        "read",
        [
            lambda: param({"key": "10"}, "key", 10),
            lambda: param({"key": 1.5}, "key", 10),
            lambda: param({"key": None}, "key", 0.0),
            lambda: param({"key": [1]}, "key", ""),
            lambda: param({"key": True}, "key", 101),
            lambda: param({"key": False}, "key", 0.0),
            lambda: param({"key": 1}, "key", False),
        ],
    )
    def test_should_reject_a_parameter_of_another_type(
        self, read: Callable[[], object]
    ) -> None:
        with pytest.raises(TypeError, match="Parameter 'key'"):
            read()
