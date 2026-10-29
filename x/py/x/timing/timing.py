#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import sys
import time
from collections.abc import Callable
from typing import Literal, TypeVar

from x.telem import CrudeTimeSpan, Rate, TimeSpan, TimeStamp

T = TypeVar("T")

# How long before a precise deadline the OS sleep hands over to a spin. Windows timers
# wake up to 0.5 ms late, so its margin is larger. A full spin is no more accurate, and
# it holds the GIL, which stalls other Python threads by milliseconds.
_SPIN_MARGIN = (
    TimeSpan.MILLISECOND if sys.platform == "win32" else 500 * TimeSpan.MICROSECOND
).nanoseconds


def _sleep_until(deadline: int, *, precise: bool) -> None:
    remaining = deadline - time.perf_counter_ns()
    if not precise:
        if remaining > 0:
            time.sleep(remaining / 1e9)
        return
    # macOS wakes up to 50% late, so each OS sleep covers only half of what is left.
    while remaining > 2 * _SPIN_MARGIN:
        time.sleep((remaining - _SPIN_MARGIN) / 2e9)
        remaining = deadline - time.perf_counter_ns()
    while time.perf_counter_ns() < deadline:
        pass


def sleep(dur: Rate | TimeSpan | float | int, *, precise: bool = False) -> None:
    """Sleeps for the given duration.

    :param dur: The duration to sleep for. A float or int is a number of seconds, and a
        Rate is one period of that rate (1 Hz = 1 second).
    :param precise: Whether to wake within microseconds of the deadline. A precise sleep
        sleeps in the OS until shortly before the deadline, then busy-waits the rest, so
        it uses more CPU than the default sleep.
    """
    span = TimeSpan.from_seconds(dur)
    if not precise:
        return time.sleep(span.seconds)
    return _sleep_until(time.perf_counter_ns() + span.nanoseconds, precise=True)


class Timer:
    """Timer is a class that implements a simple timer. It is used to measure the time
    elapsed since the timer was started. The timer uses the high-resolution performance
    counter to measure time. The timer can be started, reset, and the elapsed time can
    be queried.
    """

    _start: TimeStamp

    def __init__(self) -> None:
        self.reset()

    def elapsed(self) -> TimeSpan:
        """Returns the time elapsed since the timer was started."""
        return TimeStamp(time.perf_counter_ns()) - self._start

    def start(self) -> None:
        """Starts the timer."""
        self.reset()

    def reset(self) -> None:
        """Resets the timer to zero."""
        self._start = TimeStamp(time.perf_counter_ns())


class Loop:
    """Loop runs a block of code at a fixed rate. Iteration n is due at start + n *
    interval, so the time the block takes and late OS wake-ups do not accumulate into
    drift. The schedule starts when the loop is created or entered as a context manager.

    An iteration that starts late returns at once and keeps the schedule, so the loop
    catches up on a short overrun. An overrun of one full interval or more drops the
    missed iterations and restarts the schedule from the current time, so the block
    never runs in a burst.
    """

    interval: TimeSpan
    """The interval at which to run the loop."""
    precise: bool
    """Whether the loop uses a precise sleep."""
    counter: int
    """The number of iterations that have been run."""
    _start: int
    _last: int
    _deadline: int

    def __init__(
        self, interval: Rate | TimeSpan | float | int, *, precise: bool = False
    ):
        """Creates a new Loop object with the given interval and precision.

        :param interval: The interval at which to run the loop. A float or int is a
            number of seconds, and a Rate is one period of that rate (1 Hz = 1 second).
        :param precise: Whether to wake within microseconds of each deadline. See
            sleep for the CPU cost.
        """
        self.interval = TimeSpan.from_seconds(interval)
        self.precise = precise
        self._restart()

    def _restart(self) -> None:
        self.counter = 0
        self._start = self._last = time.perf_counter_ns()
        self._deadline = self._start + self.interval.nanoseconds

    @property
    def average(self) -> TimeSpan:
        """The average time between iterations since the schedule started."""
        if self.counter == 0:
            return TimeSpan(0)
        return TimeSpan((self._last - self._start) // self.counter)

    def __enter__(self) -> Loop:
        self._restart()
        return self

    def __exit__(self, exc_type: object, exc_val: object, exc_tb: object) -> None:
        pass

    def __iter__(self) -> Loop:
        return self

    def __next__(self) -> None:
        interval = self.interval.nanoseconds
        if time.perf_counter_ns() < self._deadline:
            _sleep_until(self._deadline, precise=self.precise)
        now = time.perf_counter_ns()
        if now - self._deadline >= interval:
            self._deadline = now
        self._deadline += interval
        self.counter += 1
        self._last = now

    def __call__(self) -> None:
        return self.__next__()

    def wait(self) -> Literal[True]:
        """Waits until the next iteration of the loop is due.

        :returns: True, so the call can be a while condition.
        """
        self.__next__()
        return True


def poll(
    func: Callable[[], T | None],
    *,
    timeout: CrudeTimeSpan,
    interval: CrudeTimeSpan = TimeSpan.SECOND,
) -> T | None:
    """Repeatedly calls func until it returns a non-None value or the timeout expires.

    :param func: A callable that returns a value or None. Polling stops when it returns
    a non-None value.
    :param timeout: Maximum time to poll for. A float or int is a number of seconds.
    :param interval: Time to sleep between calls to func. A float or int is a number of
    seconds.
    :returns: The first non-None return value from func, or None if the timeout expired.
    """
    timeout = TimeSpan.from_seconds(timeout)
    interval = TimeSpan.from_seconds(interval)
    timer = Timer()
    while timer.elapsed() < timeout:
        result = func()
        if result is not None:
            return result
        sleep(interval)
    return None
