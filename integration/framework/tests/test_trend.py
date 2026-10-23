#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from framework.trend import rise, slope

TIMES = [float(t) for t in range(0, 600, 30)]


class TestSlope:
    def test_line(self) -> None:
        """Should return the slope of samples on a line."""
        assert slope(TIMES, [5 + 0.5 * t for t in TIMES]) == pytest.approx(0.5)

    def test_spike(self) -> None:
        """Should ignore one spike in flat samples."""
        values = [100.0] * len(TIMES)
        values[len(values) // 2] = 1e6
        assert slope(TIMES, values) == 0

    def test_swing(self) -> None:
        """Should read samples that swing between two levels as flat."""
        values = [10.0 + 4 * (i % 2) for i in range(len(TIMES))]
        assert slope(TIMES, values) == pytest.approx(0, abs=0.01)

    def test_same_time(self) -> None:
        """Should skip pairs of samples taken at the same time."""
        assert slope([0, 0, 10], [0, 5, 10]) == pytest.approx(0.75)

    def test_one_time(self) -> None:
        """Should raise when no two samples differ in time."""
        with pytest.raises(ValueError, match="Need two samples taken at different"):
            slope([1, 1], [2, 3])

    def test_length_mismatch(self) -> None:
        """Should raise when times and values differ in length."""
        with pytest.raises(ValueError, match="Got 2 times but 1 values"):
            slope([1, 2], [3])


class TestRise:
    def test_leak(self) -> None:
        """Should return the growth of a steady leak over the whole span."""
        assert rise(TIMES, [2 * t for t in TIMES]) == pytest.approx(2 * TIMES[-1])

    def test_flat(self) -> None:
        """Should return zero for samples that do not change."""
        assert rise(TIMES, [7.0] * len(TIMES)) == 0
