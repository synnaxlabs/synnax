#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from typing import Callable


def assert_eventually(
    condition_func: Callable[[], any], timeout: float = 5.0, interval: float = 0.1
) -> None:
    """
    Repeatedly calls a condition function until it does not raise an AssertionError
    or the timeout is reached.

    Args:
        condition_func (Callable[[], None]): A callable that raises an AssertionError
            if the condition is not met.
        timeout (float): Maximum time (in seconds) to wait for the condition. Default is 5 seconds.
        interval (float): Time (in seconds) between retries. Default is 0.1 seconds.

    Raises:
        AssertionError: If the condition_func keeps raising an AssertionError until the timeout.
    """
    end_time = time.time() + timeout
    while time.time() < end_time:
        try:
            condition_func()
            return  # Condition succeeded, exit the function
        except:
            time.sleep(interval)
    # If the loop exits without success, call condition_func one last time to raise its error
    condition_func()
