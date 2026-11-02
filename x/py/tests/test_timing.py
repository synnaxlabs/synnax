#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import threading

import pytest

from x.telem import Rate, TimeSpan
from x.timing import Loop, Timer, poll, sleep


class FakeClock:
    """A clock that moves only when a caller reads it, sleeps on it, or advances it.
    Each read takes a fixed tick, and each sleep wakes late by a fixed lateness.
    """

    def __init__(
        self, *, lateness: TimeSpan = TimeSpan.ZERO, tick: TimeSpan = TimeSpan.ZERO
    ) -> None:
        self.time = 0
        self.lateness = lateness.nanoseconds
        self.tick = tick.nanoseconds
        self.sleeps: list[int] = []

    def now(self) -> int:
        self.time += self.tick
        return self.time

    def advance(self, span: TimeSpan) -> None:
        self.time += span.nanoseconds

    def sleep(self, span: int) -> None:
        self.sleeps.append(span)
        self.time += span + self.lateness


# A precise sleep cannot correct a wake later than 1.5 spin margins, so the lateness
# stays below that on every platform.
LATENESS = TimeSpan.MICROSECOND * 500
TICK = TimeSpan.MICROSECOND


@pytest.mark.timing
class TestTiming:
    def test_sleep(self) -> None:
        """Should wake at the deadline on a precise sleep and one lateness after it on a
        standard sleep when each OS sleep wakes late.
        """
        duration = TimeSpan.MILLISECOND * 5
        standard = FakeClock(lateness=LATENESS)
        sleep(duration, clock=standard)
        assert standard.sleeps == [duration]
        assert standard.time == duration + LATENESS
        precise = FakeClock(lateness=LATENESS, tick=TICK)
        sleep(duration, precise=True, clock=precise)
        assert duration + TICK <= precise.time < duration + TICK * 2

    def test_sleep_rate(self) -> None:
        """Should convert a rate argument into the equivalent sleep duration."""
        clock = FakeClock()
        sleep(100 * Rate.HZ, clock=clock)
        assert clock.sleeps == [TimeSpan.MILLISECOND * 10]

    @pytest.mark.parametrize("lateness", [TimeSpan.ZERO, LATENESS])
    @pytest.mark.parametrize("duration_us", [100, 3_000, 200_000])
    def test_precise_sleep_never_returns_early(
        self, lateness: TimeSpan, duration_us: int
    ) -> None:
        """Should not return before the requested duration elapses."""
        duration = TimeSpan.MICROSECOND * duration_us
        clock = FakeClock(lateness=lateness, tick=TICK)
        sleep(duration, precise=True, clock=clock)
        assert clock.time >= duration + TICK

    def test_precise_sleep_yields_cpu(self) -> None:
        """Should sleep in the OS for most of a long duration instead of spinning."""
        duration = TimeSpan.MILLISECOND * 200
        clock = FakeClock(tick=TICK)
        sleep(duration, precise=True, clock=clock)
        assert duration.nanoseconds - sum(clock.sleeps) <= TimeSpan.MILLISECOND * 2

    def test_zero_sleep_yields(self) -> None:
        """Should let another thread run on a zero sleep."""
        go = threading.Event()
        done = threading.Event()

        def run() -> None:
            go.wait()
            done.set()

        thread = threading.Thread(target=run)
        thread.start()
        interval = sys.getswitchinterval()
        # The thread can then only run when sleep releases the GIL.
        sys.setswitchinterval(1000)
        try:
            go.set()
            t = Timer()
            while not done.is_set() and t.elapsed() < TimeSpan.SECOND * 10:
                sleep(0)
            yielded = done.is_set()
        finally:
            sys.setswitchinterval(interval)
        thread.join()
        assert yielded

    def test_poll_returns_value_when_condition_met(self) -> None:
        """Should return the non-None value when the condition is met."""
        call_count = 0

        def condition() -> str | None:
            nonlocal call_count
            call_count += 1
            return "done" if call_count >= 3 else None

        clock = FakeClock()
        result = poll(
            condition,
            timeout=5 * TimeSpan.SECOND,
            interval=10 * TimeSpan.MILLISECOND,
            clock=clock,
        )
        assert result == "done"
        assert call_count == 3
        assert clock.sleeps == [TimeSpan.MILLISECOND * 10] * 2

    def test_poll_returns_none_on_timeout(self) -> None:
        """Should return None when the timeout expires."""
        call_count = 0

        def condition() -> None:
            nonlocal call_count
            call_count += 1

        clock = FakeClock()
        result = poll(
            condition,
            timeout=50 * TimeSpan.MILLISECOND,
            interval=10 * TimeSpan.MILLISECOND,
            clock=clock,
        )
        assert result is None
        assert call_count == 5
        assert clock.time == TimeSpan.MILLISECOND * 50

    def test_poll_reads_numbers_as_seconds(self) -> None:
        """Should read a numeric timeout and interval as seconds, like sleep"""
        call_count = 0

        def condition() -> str | None:
            nonlocal call_count
            call_count += 1
            return "done" if call_count >= 3 else None

        clock = FakeClock()
        assert poll(condition, timeout=5, interval=0.01, clock=clock) == "done"
        assert clock.sleeps == [TimeSpan.MILLISECOND * 10] * 2

    def test_poll_returns_immediately(self) -> None:
        """Should return immediately if the condition is already met."""
        clock = FakeClock()
        result = poll(
            lambda: True,
            timeout=5 * TimeSpan.SECOND,
            interval=TimeSpan.SECOND,
            clock=clock,
        )
        assert result is True
        assert clock.sleeps == []


@pytest.mark.timing
class TestLoop:
    def test_no_drift(self) -> None:
        """Should keep each iteration on the original schedule when the block takes a
        large part of the interval and every sleep wakes late.
        """
        interval = TimeSpan.MILLISECOND * 10
        lateness = TimeSpan.MILLISECOND
        clock = FakeClock(lateness=lateness)
        stamps: list[int] = []
        with Loop(interval, clock=clock) as loop:
            for _ in loop:
                stamps.append(clock.now())
                if len(stamps) == 200:
                    break
                clock.advance(TimeSpan.MILLISECOND * 4)
        assert stamps == [interval * (i + 1) + lateness for i in range(200)]
        assert clock.sleeps == [interval] + [TimeSpan.MILLISECOND * 5] * 199

    def test_precise_no_drift(self) -> None:
        """Should wake each iteration at its deadline on a precise loop when every sleep
        wakes late.
        """
        interval = TimeSpan.MILLISECOND * 10
        clock = FakeClock(lateness=LATENESS, tick=TICK)
        stamps: list[int] = []
        with Loop(interval, precise=True, clock=clock) as loop:
            for _ in loop:
                stamps.append(clock.now())
                if len(stamps) == 200:
                    break
                clock.advance(TimeSpan.MILLISECOND * 4)
        for i, stamp in enumerate(stamps):
            assert 0 <= stamp - (interval * (i + 1)).nanoseconds < TICK * 5

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
        clock = FakeClock()
        with Loop(interval, clock=clock) as loop:
            for _ in range(3):
                loop.wait()
        assert loop.counter == 3
        assert loop.average == interval
