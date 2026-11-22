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
from synnax.calculation.types_gen import RetrieveRequest
from synnax.util import sole


class TestSoleWithoutSearchTerm:
    """Tests for a single retrieve whose request has no search term."""

    def test_not_found(self) -> None:
        """Should describe the filter of a request that has no search term."""
        with pytest.raises(sy.NotFoundError, match=r"^Found no calculation$"):
            sole("calculation", RetrieveRequest(), [])

    def test_multiple_found(self) -> None:
        """Should count the records of a request that has no search term."""
        with pytest.raises(
            sy.MultipleFoundError,
            match=r"^Expected one calculation, but found 2$",
        ):
            sole("calculation", RetrieveRequest(), ["x", "y"])
