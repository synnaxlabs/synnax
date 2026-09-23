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
from synnax.framer.common import (
    MAX_DOWNSAMPLE_FACTOR,
    validate_downsample_factor,
)


@pytest.mark.framer
class TestValidateDownsampleFactor:
    @pytest.mark.parametrize("factor", [0, 1, 2, 10, MAX_DOWNSAMPLE_FACTOR])
    def test_accepts_in_range(self, factor: int):
        """Should accept any factor the wire field can carry."""
        validate_downsample_factor(factor)

    @pytest.mark.parametrize("factor", [-1, -2, MAX_DOWNSAMPLE_FACTOR + 1])
    def test_rejects_out_of_range(self, factor: int):
        """Should reject a factor the unsigned wire field would reinterpret."""
        with pytest.raises(sy.ValidationError):
            validate_downsample_factor(factor)
