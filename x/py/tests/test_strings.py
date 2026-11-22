#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
import sys

import numpy as np
import pytest

from x.strings import format_float


class TestFormatFloat:
    @pytest.mark.parametrize(
        ("v", "expected"),
        [
            (-1.5, "-1.5"),
            (1e10, "10000000000"),
            (1e20, "100000000000000000000"),
            (1e21, "1e+21"),
            (1e-6, "0.000001"),
            (1.5e-7, "1.5e-7"),
            (sys.float_info.max, "1.7976931348623157e+308"),
            (5e-324, "5e-324"),
            (1000000000000000128.0, "1000000000000000100"),
            (0.0, "0"),
            (-0.0, "-0"),
            (math.nan, "NaN"),
            (math.inf, "Infinity"),
            (-math.inf, "-Infinity"),
            (np.float32(0.1), "0.1"),
            (np.float32(1e20), "100000000000000000000"),
            (np.float32(1e30), "1e+30"),
            (np.float32(1e-6), "0.000001"),
            (np.float32(1.5e-7), "1.5e-7"),
            (np.float32(0.000244140625), "0.00024414062"),
            (np.float32(math.nan), "NaN"),
        ],
    )
    def test_format(self, v: float | np.floating, expected: str) -> None:
        """Should format a float as the shortest text that parses back to it."""
        assert format_float(v) == expected

    def test_round_trip(self) -> None:
        """Should format text that parses back to the same bits."""
        rng = np.random.default_rng(1)
        for v in rng.integers(0, 2**64, 20_000, dtype=np.uint64).view(np.float64):
            if not np.isnan(v):
                assert np.float64(format_float(v)).tobytes() == v.tobytes()
        for v in rng.integers(0, 2**32, 20_000, dtype=np.uint32).view(np.float32):
            if not np.isnan(v):
                assert np.float32(format_float(v)).tobytes() == v.tobytes()
