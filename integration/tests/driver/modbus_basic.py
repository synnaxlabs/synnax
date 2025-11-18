#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import modbus

from tests.driver.devices import Simulator
from tests.driver.driver import Driver



TASK_NAME: str = "Modbus Py - Read Task"
TEST_DURATION: sy.TimeSpan = 3 * sy.TimeSpan.SECOND
SAMPLE_RATE: sy.Rate = 10 * sy.Rate.HZ
STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ
CHAN_NAMES: list[str] = ["input_register_0", "input_register_1"]

class ModbusBasic(Driver):
    """
    Test Modbus TCP Basic - Read Task.

    This test:
    1. Starts the Modbus simulator server
    2. Connects to the device
    3. Creates channels for input registers
    4. Configures and runs a read task for 5 seconds
    5. Verifies data is being received
    """

    simulator: Simulator = Simulator.MODBUS

    def setup(self) -> None:
        super().setup()

        client = self.client
        dev = client.hardware.devices.retrieve(name=self.simulator.device_name)

        modbus_time = self.client.channels.create(
            name="modbus_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        input_reg_0 = self.client.channels.create(
            name=CHAN_NAMES[0],
            index=modbus_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        input_reg_1 = self.client.channels.create(
            name=CHAN_NAMES[1],
            index=modbus_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        self.tsk: modbus.ReadTask = modbus.ReadTask(
            name=TASK_NAME,
            device=dev.key,
            sample_rate=SAMPLE_RATE,
            stream_rate=STREAM_RATE,
            data_saving=True,
            channels=[
                modbus.InputRegisterChan(
                    channel=input_reg_0.key, address=0, data_type="uint8"
                ),
                modbus.InputRegisterChan(
                    channel=input_reg_1.key, address=1, data_type="uint8"
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
        """Execute the basic Modbus read task test."""
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

        self.log("Test 2 - Reconfigure Task")
        tsk.config.sample_rate = int(SAMPLE_RATE * 2)
        client.hardware.tasks.configure(tsk)
        with tsk.run():
            start_time = sy.TimeStamp.now()
            sy.sleep(TEST_DURATION)
            end_time = sy.TimeStamp.now()

        self.assert_sample_count(tsk, SAMPLE_RATE * 2, sy.TimeRange(start_time, end_time))

        self.log("Test 3 - Delete Task")
        client.hardware.tasks.delete(tsk.key)
        self.assert_task_deleted(tsk.key)

        self.log("Test 4 - Delete Device")
        client.hardware.devices.delete([device.key])
        self.assert_device_deleted(device)
