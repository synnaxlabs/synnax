#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_indexed_pair
from tests.arc.arc import ArcCase

# The wait reads a zero duration and parks. A second stage sets the duration and enters
# the first stage again, which restarts the wait.
SOURCE = """
import time
authority 200

sequence main {
    hold := i64 ns(0s)
    stage parked {
        time.wait{duration=hold} -> pw_done
        time.wait{1s} -> pw_alive
        time.wait{2s} => reenter
    }
    stage reenter {
        time.wait{10ms} -> sequence {
            hold = i64 ns(100ms)
        }
        time.wait{50ms} => parked
    }
}
pw_start_cmd => main
"""

# The message of the warning that a zero wait duration reports.
WARNING_MESSAGE = "wait duration must be positive"


class ParkedWait(ArcCase):
    """A wait whose duration variable is zero parks with a warning. The task keeps
    running, and the wait stays parked until its stage is entered again.
    """

    arc_source = SOURCE
    arc_name_prefix = "ArcParkedWait"
    start_cmd_channel = "pw_start_cmd"
    subscribe_channels = ["pw_done", "pw_alive"]

    def setup(self) -> None:
        create_indexed_pair(self.client, "pw_done", sy.DataType.UINT8)
        create_indexed_pair(self.client, "pw_alive", sy.DataType.UINT8)
        super().setup()

    def status(self) -> sy.task.Status | None:
        """Returns the current status of the Arc task."""
        task = self.client.tasks.retrieve(
            key=self.task_key(self.arc_name), include_status=True
        )
        return task.status

    def verify_sequence_execution(self) -> None:
        timer = sy.Timer()
        while (status := self.status()) is None or status.variant != "warning":
            if status is not None and status.variant == "error":
                self.fail(f"the task reports an error: {status.message}")
            if timer.elapsed() > 5 * sy.TimeSpan.SECOND:
                self.fail("the task reports no warning for the zero wait duration")
            sy.sleep(0.05)
        if WARNING_MESSAGE not in f"{status.message} {status.description}":
            self.fail(f"the warning is not about the wait: {status.message}")
        self.wait_for_eq("pw_alive", 1, timeout=5 * sy.TimeSpan.SECOND)
        if self.read_tlm("pw_done") == 1:
            self.fail("the parked wait fired before its stage was entered again")
        self.wait_for_eq("pw_done", 1, timeout=10 * sy.TimeSpan.SECOND)
        status = self.status()
        if status is not None and status.variant == "error":
            self.fail(f"the task reports an error: {status.message}")
