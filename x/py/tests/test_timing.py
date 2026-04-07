#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import numpy as np
import pytest

from x.telem import Rate, TimeSpan
from x.timing import Loop, Timer, poll, sleep


@pytest.mark.timing
class TestTiming:
    def test_sleep(self) -> None:
        """Test that the sleep function is consistently better that time.sleep in terms
        of precision.

        Execute 50 different timing tests at random intervals between 100 microseconds
        and 5 milliseconds with both time.sleep and sy.sleep.
        """
        accumulated_precise: list[int] = []
        accumulated_standard: list[int] = []
        for _ in range(50):
            duration = TimeSpan.MICROSECOND * float(np.random.uniform(100, 5_000))
            start = time.perf_counter_ns()
            sleep(duration.seconds, precise=False)
            time_elapsed = time.perf_counter_ns() - start
            start = time.perf_counter_ns()
            sleep(duration, precise=True)
            sy_elapsed = time.perf_counter_ns() - start
            standard_delta = abs(time_elapsed - duration.nanoseconds)
            precise_delta = abs(sy_elapsed - duration.nanoseconds)
            accumulated_precise.append(precise_delta)
            accumulated_standard.append(standard_delta)

        assert sum(accumulated_precise) < sum(accumulated_standard)

    def test_sleep_rate(self) -> None:
        """Should sleep correctly based on a rate argument."""
        t = Timer()
        sleep(100 * Rate.HZ, precise=True)
        assert t.elapsed() < TimeSpan.MILLISECOND * 11
        assert t.elapsed() > TimeSpan.MILLISECOND * 9

    def test_poll_returns_value_when_condition_met(self) -> None:
        """Should return the non-None value when the condition is met."""
        call_count = 0

        def condition() -> str | None:
            nonlocal call_count
            call_count += 1
            return "done" if call_count >= 3 else None

        result = poll(
            condition,
            timeout=5 * TimeSpan.SECOND,
            interval=10 * TimeSpan.MILLISECOND,
        )
        assert result == "done"
        assert call_count == 3

    def test_poll_returns_none_on_timeout(self) -> None:
        """Should return None when the timeout expires."""
        result = poll(
            lambda: None,
            timeout=50 * TimeSpan.MILLISECOND,
            interval=10 * TimeSpan.MILLISECOND,
        )
        assert result is None

    def test_poll_returns_immediately(self) -> None:
        """Should return immediately if the condition is already met."""
        t = Timer()
        result = poll(
            lambda: True,
            timeout=5 * TimeSpan.SECOND,
            interval=TimeSpan.SECOND,
        )
        assert result is True
        assert t.elapsed() < 100 * TimeSpan.MILLISECOND

    def test_loop(self) -> None:
        """Test that the loop holds timing consistent even when operations in the loop
        take a long time.
        """
        loop = Loop(TimeSpan.MILLISECOND * 10, precise=True)
        i = 0
        start = time.perf_counter_ns()
        with loop:
            for _ in loop:
                i += 1
                if i == 10:
                    break
                sleep(TimeSpan.MILLISECOND * 5, precise=True)
        end = time.perf_counter_ns()
        assert TimeSpan(end - start) < TimeSpan.MILLISECOND * 110
