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

from synnax.telem import TimeSpan, Rate

RESOLUTION = (100 * TimeSpan.MICROSECOND).seconds

def _precise_sleep(dur: float | int):
    """Sleep implements a higher precision alternative to time.sleep. It uses welford's
    algorithm to estimate the ideal time to sleep for the given duration. This function
    uses considerably more CPU than time.sleep, so it should only be used when high
    precision is required.
    """
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
    """Sleep is a function that sleeps for the given duration. The duration can be
    specified as a Rate, TimeSpan, float, or int. If the precise flag is set to True,
    the function will use a more precise sleep implementation. It uses welford's
    algorithm to estimate the ideal time to sleep for the given duration. This function
    uses considerably more CPU than time.sleep, so it should only be used when high
    precision is required.

    Args:
        dur (Rate | TimeSpan | float | int): The duration to sleep for.
        precise (bool): Whether to use a more precise sleep implementation. This will
            use more CPU than the default sleep implementation, but will provide significantly
            higher precision
    """
    dur = TimeSpan.parse_seconds(dur).seconds
    if precise:
        return _precise_sleep(dur)
    return time.sleep(dur)


class Timer:
    """Timer is a class that implements a simple timer. It is used to measure the time
    elapsed since the timer was started. The timer uses the high-resolution performance
    counter to measure time. The timer can be started, reset, and the elapsed time can
    be queried.
    """
    _start: TimeSpan

    def __init__(self):
        self.reset()

    def elapsed(self) -> TimeSpan:
        """Elapsed returns the time elapsed since the timer was started.
        """
        return TimeSpan(time.perf_counter_ns() - self._start)

    def start(self):
        """Start starts the timer.
        """
        self.reset()

    def reset(self):
        """Reset resets the timer to zero.
        """
        self._start = TimeSpan(time.perf_counter_ns())


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
    counter: int = 0
    average: TimeSpan = TimeSpan(0)
    correction: TimeSpan = TimeSpan(0)

    def __init__(
        self,
        interval: Rate | TimeSpan | float | int,
        precise: bool = False
    ):
        """Creates a new Loop object with the given interval and precision.

        Args:
            interval (Rate | TimeSpan | float | int): The interval at which to run the loop.
            precise (bool): Whether to use a more precise sleep implementation. This will
                use more CPU than the default sleep implementation, but will provide significantly
                higher precision.
        """
        self._timer = Timer()
        self.interval = TimeSpan.parse_seconds(interval)
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
            sleep_for = self.interval - elapsed - self.correction
            if sleep_for < TimeSpan(0):
                sleep_for = TimeSpan(0)
            sleep(sleep_for, self.precise)
        self.counter += 1
        self.average = TimeSpan(
            (self.average * (self.counter - 1) + self._timer.elapsed()) / self.counter
        )
        self.correction = self.average - self.interval
        self._timer.reset()

    def __call__(self):
        return self.__next__()

    def wait(self) -> True:
        """Wait is a blocking function that will wait for the next iteration of the loop.
        This function will block until the next iteration of the loop is complete.
        """
        self.__next__()
        return True

