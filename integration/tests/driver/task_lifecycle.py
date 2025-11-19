#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Task Lifecycle Test

This test validates the complete lifecycle of driver tasks across different protocols:
1. Task configuration and creation
2. Start and stop operations
3. Data acquisition and validation
4. Task reconfiguration
5. Task deletion
6. Device cleanup

Supports multiple protocols via matrix parameter:
- modbus
- opcua
- opcua_bool
- opcua_array

Example JSON configuration:
{
  "case": "driver/task_lifecycle",
  "parameters": {
    "protocol": ["modbus", "opcua", "opcua_bool", "opcua_array"]
  }
}
"""

import synnax as sy

from tests.driver.driver import Driver
from tests.driver.tasks import TASK_CASE, assemble_task

# Test constants
TEST_DURATION: sy.TimeSpan = 3 * sy.TimeSpan.SECOND
SAMPLE_RATE: sy.Rate = 10 * sy.Rate.HZ
STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ


class TaskLifecycle(Driver):
    """
    Protocol-agnostic task lifecycle test.

    This test validates complete task lifecycle across different protocols:
    - Task creation and configuration
    - Start/stop operations
    - Data acquisition
    - Task reconfiguration
    - Cleanup (task and device deletion)

    The protocol is selected via the 'protocol' parameter from the test matrix.
    """

    def setup(self) -> None:
        """Configure test based on protocol parameter."""

        # Load TaskCase From Matrix Parameter
        task = self.params.get("protocol", "None")
        self.log(f"Configuring test for task: {task}")
        if task not in TASK_CASE:
            self.fail(f"Unknown task: {task}. " f"Available: {list(TASK_CASE.keys())}")
            return

        config = TASK_CASE[task]
        if config.simulator is not None:
            self.simulator = config.simulator

        super().setup()

        self.tsk, self.channel_names = assemble_task(
            self.client,
            self.simulator.device_name,
            config,
            task,
            SAMPLE_RATE,
            STREAM_RATE,
        )

        try:
            self.client.hardware.tasks.configure(self.tsk)
            self.log(f"Task '{config.task_name}' configured")
        except Exception as e:
            self.fail(f"Task configuration failed: {e}")
            return

    def run(self) -> None:
        """Execute the task lifecycle test."""
        client = self.client
        tsk = self.tsk

        self.log("Test 0 - Verify Task Exists")
        self.assert_task_exists(tsk.key)
        self.assert_channel_names(tsk, self.channel_names)

        self.log("Test 1 - Start and Stop")
        start_time = sy.TimeStamp.now()
        with tsk.run():
            sy.sleep(TEST_DURATION)
        end_time = sy.TimeStamp.now()

        self.assert_sample_count(tsk, SAMPLE_RATE, sy.TimeRange(start_time, end_time))

        self.log("Test 2 - Reconfigure Task")
        tsk.config.sample_rate = int(SAMPLE_RATE * 2)
        client.hardware.tasks.configure(tsk)
        start_time = sy.TimeStamp.now()
        with tsk.run():
            sy.sleep(TEST_DURATION)
        end_time = sy.TimeStamp.now()

        self.assert_sample_count(
            tsk, SAMPLE_RATE * 2, sy.TimeRange(start_time, end_time)
        )

        self.log("Test 3 - Delete Task")
        client.hardware.tasks.delete(tsk.key)
        self.assert_task_deleted(tsk.key)

        self.log("Test 4 - Delete Device")
        # Get device from task config (already embedded in task object)
        device = client.hardware.devices.retrieve(key=tsk.config.device)
        client.hardware.devices.delete([device.key])
        self.assert_device_deleted(device)
