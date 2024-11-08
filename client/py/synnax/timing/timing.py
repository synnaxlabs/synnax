#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import math

from synnax.telem import TimeSpan, Rate, TimeStamp

RESOLUTION = (100 * TimeSpan.MICROSECOND).seconds


def _precise_sleep(dur: float | int):
    estimate = RESOLUTION * 10  # Initial overestimate
    mean = RESOLUTION * 10
    m2 = 0
    count = 1
    end_time = time.perf_counter() + dur
    nanoseconds = dur * 1e9
    while nanoseconds > estimate * 1e9:
        start_time = time.perf_counter()
        time.sleep(RESOLUTION)
        # Elapsed time in nanoseconds
        elapsed = (time.perf_counter() - start_time) * 1e9
        delta = elapsed - mean
        mean += delta / count
        m2 += delta * (elapsed - mean)
        estimate = mean + 1 * math.sqrt(m2 / count)
        count += 1

    # Busy wait for the last bit to ensure we sleep for the correct duration
    while time.perf_counter() < end_time:
        pass


def sleep(dur: Rate | TimeSpan | float | int, precise: bool = False):
    """Sleeps for the given duration, with the option to use a high-precision sleep
    that is more accurate than Python's default time.sleep implementation.


    :param dur: The duration to sleep for. The value can be a float or int representing
    the number of seconds to sleep for, a Rate object representing the rate at
    which to sleep (i.e. 1 Hz = 1 second), or a TimeSpan object representing the
    duration to sleep for.
    :param precise: Whether to use a more precise sleep implementation. This will
    use more CPU than the default sleep implementation, but will provide significantly
    higher precision. It uses Welford's algorithm to estimate the ideal time to sleep
    for the given duration.
    """
    dur = TimeSpan.from_seconds(dur).seconds
    if precise:
        return _precise_sleep(dur)
    return time.sleep(dur)


class Timer:
    """Timer is a class that implements a simple timer. It is used to measure the time
    elapsed since the timer was started. The timer uses the high-resolution performance
    counter to measure time. The timer can be started, reset, and the elapsed time can
    be queried.
    """

    _start: TimeStamp

    def __init__(self):
        self.reset()

    def elapsed(self) -> TimeSpan:
        """Returns the time elapsed since the timer was started."""
        return TimeSpan(time.perf_counter_ns() - self._start)

    def start(self):
        """Starts the timer."""
        self.reset()

    def reset(self):
        """Resets the timer to zero."""
        self._start = TimeStamp(time.perf_counter_ns())


class Loop:
    """Loop is a class that implements a rate-limited loop. It is used to ensure that a
    block of code is executed at a fixed rate. The loop will sleep for the remainder of
    the interval if the block of code executes faster than the interval. If the block of
    code takes longer than the interval, the loop will skip sleeping to ensure that the
    block of code is executed as soon as possible. The loop keeps track of the average
    execution time and uses this to correct for any drift in the interval.
    """

    _timer: Timer
    interval: TimeSpan
    """The interval at which to run the loop."""
    counter: int = 0
    """The number of iterations that have been run."""
    average: TimeSpan = TimeSpan(0)
    """The average execution time of the loop."""
    _correction: TimeSpan = TimeSpan(0)

    def __init__(self, interval: Rate | TimeSpan | float | int, precise: bool = False):
        """Creates a new Loop object with the given interval and precision.

        :param interval: The interval at which to run the loop. This can be a float or
                int representing the number of seconds between each iteration, a Rate
                object representing the rate at which to run the loop (i.e. 1 Hz = 1
                second per iteration), or a TimeSpan object representing the duration
                between each iteration.
        :param precise: Whether to use a more precise sleep implementation. This will
                use more CPU than the default sleep implementation, but will provide
                significantly higher precision. It uses Welford's algorithm to estimate
                the ideal time to sleep for the given duration.
        """
        self._timer = Timer()
        self.interval = TimeSpan.from_seconds(interval)
        self.precise = precise
        self._last = time.perf_counter_ns()

    def __enter__(self):
        self._last = time.perf_counter_ns()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    def __iter__(self):
        return self

    def __next__(self):
        elapsed = self._timer.elapsed()
        if elapsed < self.interval:
            sleep_for = self.interval - elapsed - self._correction
            if sleep_for < TimeSpan(0):
                sleep_for = TimeSpan(0)
            sleep(sleep_for, self.precise)
        self.counter += 1
        self.average = TimeSpan(
            (self.average * (self.counter - 1) + self._timer.elapsed()) / self.counter
        )
        self._correction = self.average - self.interval
        self._timer.reset()

    def __call__(self):
        return self.__next__()

    def wait(self) -> True:
        """Waits for the next iteration of the loop, automatically sleeping for the
        remainder of the interval if the calling block of code executes faster than the
        interval.
        """
        self.__next__()
        return True
