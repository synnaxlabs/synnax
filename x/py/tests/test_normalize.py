#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from x.normalize import check_for_none, normalize, override


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

    def test_multiple_args(self) -> None:
        assert normalize(1, [2, 3], 4) == [1, 2, 3, 4]


class TestCheckForNone:
    def test_all_none(self) -> None:
        assert check_for_none(None, None) is True

    def test_some_not_none(self) -> None:
        assert check_for_none(None, 1) is False

    def test_none_not_present(self) -> None:
        assert check_for_none(1, 2) is False


class TestOverride:
    def test_returns_first_non_none(self) -> None:
        result: list[Any] | None = override(None, [1, 2])
        assert result == [1, 2]

    def test_all_none_returns_none(self) -> None:
        assert override(None, None) is None

    def test_normalizes_result(self) -> None:
        assert override(5) == [5]
