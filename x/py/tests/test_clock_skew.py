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
        mock_time[0] = TimeStamp(TimeSpan.SECOND * 10)
        # Remote midpoint is 3s, local midpoint is 5s, so skew is 2s
        calc.end(TimeStamp(TimeSpan.SECOND * 3))
        assert calc.skew == TimeSpan.SECOND * 2
        assert calc.exceeds(TimeSpan.SECOND) is True
        assert calc.exceeds(TimeSpan.SECOND * 3) is False

    def test_zero_skew(self) -> None:
        mock_time = [TimeStamp(0)]

        def now() -> TimeStamp:
            return mock_time[0]

        calc = ClockSkewCalculator(now=now)
        calc.start()
        mock_time[0] = TimeStamp(TimeSpan.SECOND * 10)
        # Remote midpoint matches local midpoint at 5s
        calc.end(TimeStamp(TimeSpan.SECOND * 5))
        assert calc.skew == TimeSpan(0)
        assert calc.exceeds(TimeSpan.SECOND) is False

    def test_returns_most_recent_measurement(self) -> None:
        mock_time = [TimeStamp(0)]

        def now() -> TimeStamp:
            return mock_time[0]

        calc = ClockSkewCalculator(now=now)
        calc.start()
        mock_time[0] = TimeStamp(TimeSpan.SECOND * 10)
        calc.end(TimeStamp(TimeSpan.SECOND * 3))
        assert calc.skew == TimeSpan.SECOND * 2
        mock_time[0] = TimeStamp(0)
        calc.start()
        mock_time[0] = TimeStamp(TimeSpan.SECOND * 10)
        # Remote midpoint is 7s, local midpoint is 5s, so skew is -2s
        calc.end(TimeStamp(TimeSpan.SECOND * 7))
        assert calc.skew == TimeSpan.SECOND * -2

    def test_no_measurements(self) -> None:
        calc = ClockSkewCalculator()
        assert calc.skew == TimeSpan(0)
        assert calc.exceeds(TimeSpan.SECOND) is False
