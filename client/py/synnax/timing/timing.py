import time
import math

import synnax
from synnax.telem import TimeSpan

RESOLUTION = 100 * 1e-6  # Resolution in seconds (100 microseconds)


def sleep(dur: TimeSpan | float | int):
    """Sleep implements a higher precision alternative to time.sleep. It uses welford's
    algorithm to estimate the ideal time to sleep for the given duration. This function
    uses considerably more CPU than time.sleep, so it should only be used when high
    precision is required.
    """
    if isinstance(dur, TimeSpan):
        dur = dur.seconds
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
