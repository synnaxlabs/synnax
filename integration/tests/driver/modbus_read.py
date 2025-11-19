#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Modbus Read Task Test

This test validates the standard task lifecycle for Modbus TCP read tasks.
Configuration is self-contained within this test class.

Example JSON configuration:
{
  "case": "driver/modbus_read"
}
"""

import synnax as sy
from synnax.hardware import modbus

from tests.driver.devices import Simulator
from tests.driver.task import ChannelConfig, Task


class ModbusRead(Task):
    """
    Modbus TCP read task lifecycle test.

    This test configures and validates a standard Modbus read task with
    input register channels. The task configuration is self-contained.
    """

    # Test configuration - see client/py/examples/modbus/server.py
    simulator = Simulator.MODBUS
    TASK_NAME = "Modbus Py - Read Task"
    TASK_KEY = "modbus_read"
    CHANNELS = [
        {
            "name": "input_register_0",
            "data_type": sy.DataType.UINT8,
            "address": 0,
            "modbus_data_type": "uint8",
        },
        {
            "name": "input_register_1",
            "data_type": sy.DataType.UINT8,
            "address": 1,
            "modbus_data_type": "uint8",
        },
    ]

    def create_task(
        self,
        device: sy.Device,
        channels: list[sy.Channel],
        channel_metadata: list[ChannelConfig],
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> modbus.ReadTask:
        """
        Create a Modbus read task from channel metadata.

        Args:
            device: Synnax device to read from
            channels: Created Synnax channels
            channel_metadata: List of dicts with protocol-specific config
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured Modbus read task
        """
        task_channels = [
            modbus.InputRegisterChan(
                channel=ch.key,
                address=meta["address"],
                data_type=meta["modbus_data_type"],
            )
            for ch, meta in zip(channels, channel_metadata)
        ]

        return modbus.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=task_channels,
        )
