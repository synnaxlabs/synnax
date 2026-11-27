#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

from x.spatial import ClosedBounds


class TestClosedBounds:
    def test_parse_ordered_bounds(self) -> None:
        """Should parse bounds whose lower is below upper."""
        bounds = ClosedBounds(lower=0, upper=10)
        assert (bounds.lower, bounds.upper) == (0, 10)

    @pytest.mark.parametrize(("lower", "upper"), [(5, 5), (10, 0)])
    def test_refuse_lower_not_below_upper(self, lower: float, upper: float) -> None:
        """Should refuse bounds whose lower is not below upper."""
        with pytest.raises(ValidationError, match="lower: must be less than upper"):
            ClosedBounds(lower=lower, upper=upper)
