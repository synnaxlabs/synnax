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


class FakeClock:
    """A clock that moves only when a caller sleeps on it or advances it. Each sleep
    wakes late by a fixed lateness.
    """

    def __init__(self, *, lateness: TimeSpan = TimeSpan.ZERO) -> None:
        self.time = 0
        self.lateness = lateness.nanoseconds
        self.sleeps: list[bool] = []

    def now(self) -> int:
        return self.time

    def advance(self, span: TimeSpan) -> None:
        self.time += span.nanoseconds

    def sleep_until(self, deadline: int, *, precise: bool) -> None:
        self.sleeps.append(precise)
        self.time = max(self.time, deadline + self.lateness)


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

    @pytest.mark.parametrize("precise", [False, True])
    def test_sleep_rate(self, precise: bool) -> None:
        """Should convert a rate argument into the equivalent sleep duration."""
        clock = FakeClock()
        sleep(100 * Rate.HZ, precise=precise, clock=clock)
        assert clock.time == TimeSpan.MILLISECOND * 10
        assert clock.sleeps == [precise]

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
        """Should keep each iteration on the original schedule when the block takes a
        large part of the interval and every sleep wakes late.
        """
        interval = TimeSpan.MILLISECOND * 10
        lateness = TimeSpan.MILLISECOND
        clock = FakeClock(lateness=lateness)
        stamps: list[int] = []
        with Loop(interval, precise=precise, clock=clock) as loop:
            for _ in loop:
                stamps.append(clock.now())
                if len(stamps) == 200:
                    break
                clock.advance(TimeSpan.MILLISECOND * 4)
        assert stamps == [interval * (i + 1) + lateness for i in range(200)]
        assert clock.sleeps == [precise] * 200

    def test_catch_up_short_overrun(self) -> None:
        """Should return at once after an overrun shorter than the interval and keep the
        original schedule.
        """
        interval = TimeSpan.MILLISECOND * 200
        clock = FakeClock()
        with Loop(interval, clock=clock) as loop:
            loop.wait()
            assert clock.now() == interval
            clock.advance(interval * 1.5)
            loop.wait()
            assert clock.now() == interval * 2.5
            assert len(clock.sleeps) == 1
            loop.wait()
            assert clock.now() == interval * 3

    def test_skip_missed_iterations(self) -> None:
        """Should drop the iterations missed by an overrun of more than one interval
        instead of running them in a burst.
        """
        interval = TimeSpan.MILLISECOND * 20
        clock = FakeClock()
        with Loop(interval, clock=clock) as loop:
            loop.wait()
            clock.advance(interval * 3)
            loop.wait()
            assert clock.now() == interval * 4
            loop.wait()
            assert clock.now() == interval * 5

    def test_restart_schedule_on_enter(self) -> None:
        """Should start the schedule when the loop is entered instead of created."""
        interval = TimeSpan.MILLISECOND * 10
        clock = FakeClock()
        loop = Loop(interval, clock=clock)
        clock.advance(interval * 5)
        with loop:
            loop.wait()
        assert clock.now() == interval * 6
        assert loop.counter == 1
        assert loop.average == interval

    def test_count_iterations(self) -> None:
        """Should count each iteration and report the average time between them."""
        interval = TimeSpan.MILLISECOND * 5
        with Loop(interval) as loop:
            for _ in range(3):
                loop.wait()
        assert loop.counter == 3
        assert loop.average >= interval
