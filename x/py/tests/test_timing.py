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
        """Should convert a rate argument into the equivalent sleep duration."""
        samples: list[int] = []
        for _ in range(10):
            t = Timer()
            sleep(100 * Rate.HZ, precise=True)
            samples.append(t.elapsed())
        median = TimeSpan(int(np.median(samples)))
        assert TimeSpan.MILLISECOND * 5 < median < TimeSpan.MILLISECOND * 20

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

    def test_poll_reads_numbers_as_seconds(self) -> None:
        """Should read a numeric timeout and interval as seconds, like sleep"""
        call_count = 0

        def condition() -> str | None:
            nonlocal call_count
            call_count += 1
            return "done" if call_count >= 3 else None

        t = Timer()
        assert poll(condition, timeout=5, interval=0.01) == "done"
        assert t.elapsed() >= 20 * TimeSpan.MILLISECOND

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

    def test_precise_sleep_never_returns_early(self) -> None:
        """Should not return before the requested duration elapses."""
        for _ in range(20):
            start = time.perf_counter_ns()
            sleep(TimeSpan.MILLISECOND * 3, precise=True)
            assert time.perf_counter_ns() - start >= TimeSpan.MILLISECOND * 3

    def test_precise_sleep_yields_cpu(self) -> None:
        """Should sleep in the OS for most of a long duration instead of spinning."""
        cpu_start = time.process_time()
        sleep(TimeSpan.MILLISECOND * 200, precise=True)
        assert time.process_time() - cpu_start < 0.1


@pytest.mark.timing
class TestLoop:
    @pytest.mark.parametrize("precise", [False, True])
    def test_no_drift(self, precise: bool) -> None:
        """Should hold the average rate over many iterations when the block takes a
        large part of the interval.
        """
        interval = TimeSpan.MILLISECOND * 10
        count = 200
        stamps: list[int] = []
        start = time.perf_counter_ns()
        with Loop(interval, precise=precise) as loop:
            for _ in loop:
                stamps.append(time.perf_counter_ns() - start)
                if len(stamps) == count:
                    break
                sleep(TimeSpan.MILLISECOND * 4, precise=True)
        for i, stamp in enumerate(stamps):
            assert stamp >= interval * (i + 1)
        # A loop that slept one interval after the block would take 40% longer. The
        # slack leaves room for stalls on a loaded CI runner.
        assert stamps[-1] < interval * count * 1.2

    def test_catch_up_short_overrun(self) -> None:
        """Should return at once after an overrun shorter than the interval and keep the
        original schedule.
        """
        # A schedule restarted after the overrun would end half an interval later, so
        # the interval is long enough that a CI stall stays inside that margin.
        interval = TimeSpan.MILLISECOND * 200
        start = time.perf_counter_ns()
        with Loop(interval, precise=True) as loop:
            loop.wait()
            sleep(interval * 1.5, precise=True)
            before = time.perf_counter_ns()
            loop.wait()
            caught_up = time.perf_counter_ns() - before
            loop.wait()
            elapsed = time.perf_counter_ns() - start
        assert caught_up < interval * 0.5
        assert interval * 3 <= elapsed < interval * 3.5

    def test_skip_missed_iterations(self) -> None:
        """Should drop the iterations missed by an overrun of more than one interval
        instead of running them in a burst.
        """
        interval = TimeSpan.MILLISECOND * 20
        with Loop(interval, precise=True) as loop:
            loop.wait()
            sleep(interval * 3, precise=True)
            loop.wait()
            before = time.perf_counter_ns()
            loop.wait()
            assert time.perf_counter_ns() - before >= interval * 0.9

    def test_count_iterations(self) -> None:
        """Should count each iteration and report the average time between them."""
        interval = TimeSpan.MILLISECOND * 5
        with Loop(interval) as loop:
            for _ in range(3):
                loop.wait()
        assert loop.counter == 3
        assert loop.average >= interval
