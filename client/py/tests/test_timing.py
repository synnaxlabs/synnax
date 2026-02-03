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

import synnax as sy


@pytest.mark.timing
class TestTiming:
    def test_sleep(self):
        """Test that the sleep function is consistently better that time.sleep in terms
        of precision.

        Execute 50 different timing tests at random intervals between 100 microseconds
        and 5 milliseconds with both time.sleep and sy.sleep.
        """
        accumulated_precise = list()
        accumulated_standard = list()
        for _ in range(50):
            duration = sy.TimeSpan.MICROSECOND * float(np.random.uniform(100, 5_000))
            start = time.perf_counter_ns()
            sy.sleep(duration.seconds, precise=False)
            time_elapsed = time.perf_counter_ns() - start
            start = time.perf_counter_ns()
            sy.sleep(duration, precise=True)
            sy_elapsed = time.perf_counter_ns() - start
            standard_delta = abs(time_elapsed - duration.nanoseconds)
            precise_delta = abs(sy_elapsed - duration.nanoseconds)
            accumulated_precise.append(precise_delta)
            accumulated_standard.append(standard_delta)

        assert sum(accumulated_precise) < sum(accumulated_standard)

    def test_sleep_rate(self):
        """Should sleep correctly based on a rate argument"""
        t = sy.Timer()
        sy.sleep(100 * sy.Rate.HZ, precise=True)
        print((100 * sy.Rate.HZ).period)
        assert t.elapsed() < sy.TimeSpan.MILLISECOND * 11
        assert t.elapsed() > sy.TimeSpan.MILLISECOND * 9

    def test_loop(self):
        """Test that the loop holds timing consistent even when operations in the loop
        take a long time.
        """
        loop = sy.Loop(sy.TimeSpan.MILLISECOND * 10, precise=True)
        i = 0
        start = time.perf_counter_ns()
        with loop:
            for _ in loop:
                i += 1
                if i == 10:
                    break
                sy.sleep(sy.TimeSpan.MILLISECOND * 5, precise=True)
        end = time.perf_counter_ns()
        assert sy.TimeSpan(end - start) < sy.TimeSpan.MILLISECOND * 110
