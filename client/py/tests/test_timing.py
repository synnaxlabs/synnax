#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
import time
import numpy as np


@pytest.mark.timing
class TestTiming:
    def test_sleep(self):
        """
        Test that the sleep function is consistently better that time.sleep in terms
        of precision.

        Execute 100 different timing tests at random intervals between 100 microseconds
        and 5 milliseconds with both time.sleep and sy.sleep.
        """
        for _ in range(20):
            duration = (sy.TimeSpan.MICROSECOND * float(np.random.uniform(100, 5_000)))
            start = time.perf_counter_ns()
            time.sleep(duration.seconds)
            time_elapsed = time.perf_counter_ns() - start
            start = time.perf_counter_ns()
            sy.sleep(duration)
            sy_elapsed = time.perf_counter_ns() - start
            time_elapsed_delta = abs(time_elapsed - duration.nanoseconds)
            sy_elapsed_delta = abs(sy_elapsed - duration.nanoseconds)
            assert sy_elapsed_delta < time_elapsed_delta
