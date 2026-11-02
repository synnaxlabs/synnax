#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from synnax.util import one, one_or_none, reject_page


class TestRejectPage:
    """Tests for the check that a retrieve of one record takes no page."""

    @pytest.mark.parametrize(("limit", "offset"), [(1, 0), (None, 1), (1, 1)])
    def test_page(self, limit: int | None, offset: int) -> None:
        """Should raise ValueError when one has a limit or an offset."""
        with pytest.raises(ValueError, match="one cannot be combined"):
            reject_page(True, limit, offset)

    def test_no_page(self) -> None:
        """Should accept one without a limit or an offset."""
        reject_page(True, None, 0)

    def test_list(self) -> None:
        """Should accept a page on a retrieve that returns a list."""
        reject_page(False, 1, 1)


class TestOne:
    """Tests for the helpers that return the one record of a single retrieve."""

    def test_one(self) -> None:
        """Should return the only record."""
        assert one("Range", "a", ["x"]) == "x"

    def test_one_not_found(self) -> None:
        """Should raise NotFoundError when no record matched."""
        with pytest.raises(sy.NotFoundError, match="Range matching a not found"):
            one("Range", "a", [])

    def test_one_multiple_found(self) -> None:
        """Should raise MultipleFoundError when more than one record matched."""
        with pytest.raises(
            sy.MultipleFoundError,
            match="Expected one Range matching a, but found 2",
        ):
            one("Range", "a", ["x", "y"])

    def test_one_or_none(self) -> None:
        """Should return the only record, or None when no record matched."""
        assert one_or_none("Range", "a", ["x"]) == "x"
        assert one_or_none("Range", "a", []) is None

    def test_one_or_none_multiple_found(self) -> None:
        """Should raise MultipleFoundError when more than one record matched."""
        with pytest.raises(sy.MultipleFoundError):
            one_or_none("Range", "a", ["x", "y"])
