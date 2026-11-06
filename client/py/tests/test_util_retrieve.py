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
from synnax.util import sole, sole_or_none


class TestSole:
    """Tests for the helpers that return the one record of a single retrieve."""

    def test_sole(self) -> None:
        """Should return the only record."""
        assert sole("Range", "a", ["x"]) == "x"

    def test_sole_not_found(self) -> None:
        """Should raise NotFoundError when no record matched."""
        with pytest.raises(sy.NotFoundError, match="Range matching a not found"):
            sole("Range", "a", [])

    def test_sole_multiple_found(self) -> None:
        """Should raise MultipleFoundError when more than one record matched."""
        with pytest.raises(
            sy.MultipleFoundError,
            match="Expected one Range matching a, but found 2",
        ):
            sole("Range", "a", ["x", "y"])

    def test_sole_or_none(self) -> None:
        """Should return the only record, or None when no record matched."""
        assert sole_or_none("Range", "a", ["x"]) == "x"
        assert sole_or_none("Range", "a", []) is None

    def test_sole_or_none_multiple_found(self) -> None:
        """Should raise MultipleFoundError when more than one record matched."""
        with pytest.raises(sy.MultipleFoundError):
            sole_or_none("Range", "a", ["x", "y"])
