#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Generator

from pydantic import ValidationError

import synnax as sy
from tests.driver.opcua_read import OPCUAReadArray

RATE_LIMIT_SECONDS = 5
RATE_LIMIT = RATE_LIMIT_SECONDS * sy.TimeSpan.SECOND


def _warnings_for_task(
    frame: sy.Frame, task_key: int
) -> Generator[sy.task.Status, None, None]:
    if "sy_status_set" not in frame:
        return
    for i in range(len(frame["sy_status_set"])):
        try:
            status = sy.task.Status.model_validate(frame["sy_status_set"][i])
        except ValidationError:
            continue
        if (
            status.details is not None
            and status.details.task == task_key
            and status.variant == "warning"
        ):
            yield status


class StatusRateLimit(OPCUAReadArray):
    """
    Status rate-limit integration test.

    Verifies that the driver's StatusHandler rate limiter (5-second window) works
    end-to-end. Inherits OPCUAReadArray but overrides array_size to 2 so the task
    expects 2-element arrays while the sim serves 5, producing a persistent
    "array too large" warning every read cycle. Asserts that no identical warning
    message repeats within 5 seconds.
    """

    array_size = 2

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
            return

        task_key = self.tsk.key
        self.log(f"Listening for warnings on task key={task_key}")
        seen: dict[str, sy.TimeStamp] = {}
        warning_count = 0
        with self.client.open_streamer(["sy_status_set"]) as streamer:
            with self.tsk.run():
                # Wait for the task to start producing data before beginning
                # the timed window. On slow CI machines, setup overhead can
                # consume most of a fixed window if the timer starts earlier.
                sy.sleep(2)
                timer = sy.Timer()
                while timer.elapsed() < 10 * sy.TimeSpan.SECOND:
                    frame = streamer.read(timeout=1)
                    if frame is None:
                        continue
                    for status in _warnings_for_task(frame, task_key):
                        warning_count += 1
                        msg = status.message
                        if msg not in seen:
                            seen[msg] = status.time
                            continue
                        gap = status.time - seen[msg]
                        if gap < RATE_LIMIT:
                            self.fail(
                                f"Duplicate warning after "
                                f"{gap / sy.TimeSpan.SECOND:.2f}s "
                                f"(expected >= {RATE_LIMIT_SECONDS}s): "
                                f"{msg[:80]}"
                            )
                            return
                        seen[msg] = status.time
        if warning_count == 0:
            self.fail(
                "No warnings received from task in 10s. "
                "Expected 'array too large/small' warnings due to array_size=2 "
                "vs sim default of 5. Rate-limiter test was not exercised."
            )
            return
        self.log(
            f"Received {warning_count} warnings, "
            f"no duplicates within {RATE_LIMIT_SECONDS}s"
        )
