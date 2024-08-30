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

from synnax.telem import TimeSpan

RESOLUTION = 100 * 1e-6  # Resolution in seconds (100 microseconds)


def _precise_sleep(dur: TimeSpan | float | int):
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


def sleep(dur: TimeSpan | float | int, precise: bool = False):
    if precise:
        return _precise_sleep(dur)
    return time.sleep(TimeSpan.parse_seconds(dur).seconds)
