#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.telem import TimeSpan, TimeStamp
from x.telem.clock_skew import ClockSkewCalculator


class TestClockSkewCalculator:
    def test_single_measurement(self) -> None:
        mock_time = [TimeStamp(0)]

        def now() -> TimeStamp:
            return mock_time[0]

        calc = ClockSkewCalculator(now=now)
        calc.start()
        mock_time[0] = TimeStamp(10)
        calc.end(TimeStamp(3))
        assert int(calc.skew) == 2
        assert calc.exceeds(TimeSpan(1)) is True
        assert calc.exceeds(TimeSpan(3)) is False

    def test_zero_skew(self) -> None:
        mock_time = [TimeStamp(0)]

        def now() -> TimeStamp:
            return mock_time[0]

        calc = ClockSkewCalculator(now=now)
        calc.start()
        mock_time[0] = TimeStamp(1000)
        calc.end(TimeStamp(500))
        assert int(calc.skew) == 0
        assert calc.exceeds(TimeSpan(1)) is False

    def test_average_multiple_measurements(self) -> None:
        mock_time = [TimeStamp(0)]

        def now() -> TimeStamp:
            return mock_time[0]

        calc = ClockSkewCalculator(now=now)
        calc.start()
        mock_time[0] = TimeStamp(10)
        calc.end(TimeStamp(3))
        mock_time[0] = TimeStamp(0)
        calc.start()
        mock_time[0] = TimeStamp(10)
        calc.end(TimeStamp(7))
        assert int(calc.skew) == 0

    def test_no_measurements(self) -> None:
        calc = ClockSkewCalculator()
        assert int(calc.skew) == 0
        assert calc.exceeds(TimeSpan(1)) is False
