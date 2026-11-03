#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Annotated

import pytest
from pydantic import TypeAdapter, ValidationError

from x import strings

AT_LEAST_4 = TypeAdapter(Annotated[str, strings.min_length(4)])
AT_MOST_4 = TypeAdapter(Annotated[str, strings.max_length(4)])


class TestUTF16Length:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [("", 0), ("abcd", 4), ("éééé", 4), ("😀😀", 4), ("a😀", 3)],
    )
    def test_length(self, value: str, expected: int) -> None:
        """Should count UTF-16 code units."""
        assert strings.utf16_length(value) == expected


class TestMinLength:
    @pytest.mark.parametrize("value", ["abcd", "éééé", "😀😀"])
    def test_valid(self, value: str) -> None:
        """Should accept a string at the minimum length."""
        assert AT_LEAST_4.validate_python(value) == value

    @pytest.mark.parametrize("value", ["abc", "ééé", "a😀"])
    def test_invalid(self, value: str) -> None:
        """Should reject a string below the minimum length."""
        with pytest.raises(ValidationError, match="must be at least 4 characters"):
            AT_LEAST_4.validate_python(value)


class TestMaxLength:
    @pytest.mark.parametrize("value", ["abcd", "éééé", "😀😀"])
    def test_valid(self, value: str) -> None:
        """Should accept a string at the maximum length."""
        assert AT_MOST_4.validate_python(value) == value

    @pytest.mark.parametrize("value", ["abcde", "ééééé", "😀😀a"])
    def test_invalid(self, value: str) -> None:
        """Should reject a string above the maximum length."""
        with pytest.raises(ValidationError, match="must be at most 4 characters"):
            AT_MOST_4.validate_python(value)
