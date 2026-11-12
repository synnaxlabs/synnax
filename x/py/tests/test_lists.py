#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from x.lists import normalize


class TestNormalize:
    def test_single_value(self) -> None:
        assert normalize(1) == [1]

    def test_list(self) -> None:
        result: list[Any] = normalize([1, 2, 3])
        assert result == [1, 2, 3]

    def test_tuple(self) -> None:
        result: list[Any] = normalize((1, 2))
        assert result == [1, 2]

    def test_none_returns_empty(self) -> None:
        assert normalize(None) == []

    def test_leading_none_skipped(self) -> None:
        assert normalize(None, 1, [2]) == [1, 2]

    def test_multiple_args(self) -> None:
        assert normalize(1, [2, 3], 4) == [1, 2, 3, 4]
