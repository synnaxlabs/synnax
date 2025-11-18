#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import opcua

from tests.driver.devices import Simulator
from tests.driver.driver import Driver


TASK_NAME: str = "OPC UA Py - Read Task"
TEST_DURATION: sy.TimeSpan = 3 * sy.TimeSpan.SECOND
SAMPLE_RATE: sy.Rate = 10 * sy.Rate.HZ
STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ
CHAN_NAMES: list[str] = ["my_float_0", "my_float_1"]


class OPCUABasic(Driver):
    """
    Test OPC UA Basic - Read Task.

    This test:
    1. Starts the OPC UA simulator server
    2. Connects to the device
    3. Creates channels for OPC UA nodes
    4. Configures and runs a read task
    5. Verifies data is being received
    """

    simulator: Simulator = Simulator.OPCUA

    def setup(self) -> None:
        super().setup()

        client = self.client
        dev = client.hardware.devices.retrieve(name=self.simulator.device_name)

        # Create index channel for timestamps
        opcua_time = self.client.channels.create(
            name="opcua_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        # Create channels for OPC UA float nodes (server provides sine wave data)
        my_float_0 = self.client.channels.create(
            name=CHAN_NAMES[0],
            index=opcua_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        my_float_1 = self.client.channels.create(
            name=CHAN_NAMES[1],
            index=opcua_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        # Store the task object for use in run()
        self.tsk: opcua.ReadTask = opcua.ReadTask(
            name=TASK_NAME,
            device=dev.key,
            sample_rate=SAMPLE_RATE,
            stream_rate=STREAM_RATE,
            data_saving=True,
            channels=[
                opcua.ReadChannel(
                    channel=my_float_0.key,
                    node_id="NS=2;I=8",  # my_float_0
                    data_type="float32",
                ),
                opcua.ReadChannel(
                    channel=my_float_1.key,
                    node_id="NS=2;I=9",  # my_float_1
                    data_type="float32",
                ),
            ],
        )

        try:
            client.hardware.tasks.configure(self.tsk)
            self.log("Task configured successfully")
        except Exception as e:
            self.fail(f"Task configuration failed: {e}")
            return

    def run(self) -> None:
        """Execute the basic OPC UA read task test."""
        client = self.client
        tsk = self.tsk
        device = client.hardware.devices.retrieve(name=self.simulator.device_name)

        self.log("Test 0 - Verify Task Exists")
        self.assert_task_exists(tsk.key)
        self.assert_channel_names(tsk, CHAN_NAMES)

        self.log("Test 1 - Start and Stop")
        with tsk.run():
            start_time = sy.TimeStamp.now()
            sy.sleep(TEST_DURATION)
            end_time = sy.TimeStamp.now()

        self.assert_sample_count(tsk, SAMPLE_RATE, sy.TimeRange(start_time, end_time))

        self.log("Test 2 - Reconfigure with auto-start")
        tsk.config.sample_rate = int(SAMPLE_RATE * 2)
        client.hardware.tasks.configure(tsk)
        with tsk.run():
            start_time = sy.TimeStamp.now()
            sy.sleep(TEST_DURATION)
            end_time = sy.TimeStamp.now()

        self.assert_sample_count(tsk, SAMPLE_RATE*2, sy.TimeRange(start_time, end_time))

        self.log("Test 3 - Delete Task")
        client.hardware.tasks.delete(tsk.key)
        self.assert_task_deleted(tsk.key)

        self.log("Test 4 - Delete Device")
        client.hardware.devices.delete([device.key])
        self.assert_device_deleted(device)