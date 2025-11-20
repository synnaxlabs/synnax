#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from typing import TypedDict

import synnax as sy
from synnax.hardware import modbus

from driver.devices import Simulator
from driver.driver import ChannelConfig
from tests.driver.task import TaskCase


class TaskTypeConfig(TypedDict, total=False):
    """
    Configuration for a specific Modbus task type.

    Attributes:
        task_name: Human-readable name for the task
        task_key: Unique identifier for the task (used for index channel naming)
        channels: List of channel configurations with Modbus-specific metadata
    """

    task_name: str
    task_key: str
    channels: list[ChannelConfig]


class ModbusRead(TaskCase):
    """
    Modbus TCP read task lifecycle test.

    This test configures and validates Modbus read tasks with different channel types.
    The task type is selected via the 'task' matrix parameter.

    Supported task types:
    - "input_register": Input register channels (function code 4, read-only)
    - "holding_register": Holding register input channels (function code 3, read/write)
    - "discrete_input": Discrete input channels (function code 2, 1-bit read-only)
    - "coil": Coil input channels (function code 1, 1-bit read/write)
    """

    # Simulator configuration
    simulator = Simulator.MODBUS

    # Task configurations for different channel types
    # See client/py/examples/modbus/server.py for simulator details
    TASK_CONFIGS: dict[str, TaskTypeConfig] = {
        "input_register": {
            "task_name": "Modbus Py - Read Input Register",
            "task_key": "modbus_read_input_register",
            "channels": [
                {
                    "name": "input_register_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "input_register",
                },
                {
                    "name": "input_register_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "input_register",
                },
            ],
        },
        "holding_register": {
            "task_name": "Modbus Py - Read Holding Register",
            "task_key": "modbus_read_holding_register",
            "channels": [
                {
                    "name": "holding_register_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "holding_register",
                },
                {
                    "name": "holding_register_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "holding_register",
                },
            ],
        },
        "discrete_input": {
            "task_name": "Modbus Py - Read Discrete Input",
            "task_key": "modbus_read_discrete_input",
            "channels": [
                {
                    "name": "discrete_input_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_channel_type": "discrete_input",
                },
                {
                    "name": "discrete_input_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_channel_type": "discrete_input",
                },
            ],
        },
        "coil": {
            "task_name": "Modbus Py - Read Coil",
            "task_key": "modbus_read_coil",
            "channels": [
                {
                    "name": "coil_input_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_channel_type": "coil",
                },
                {
                    "name": "coil_input_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_channel_type": "coil",
                },
            ],
        },
        "mixed": {
            "task_name": "Modbus Py - Read Mixed",
            "task_key": "modbus_read_mixed",
            "channels": [
                {
                    "name": "input_register_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "input_register",
                },
                {
                    "name": "input_register_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_data_type": "uint8",
                    "modbus_channel_type": "input_register",
                },
                {
                    "name": "discrete_input_0",
                    "data_type": sy.DataType.UINT8,
                    "address": 0,
                    "modbus_channel_type": "discrete_input",
                },
                {
                    "name": "discrete_input_1",
                    "data_type": sy.DataType.UINT8,
                    "address": 1,
                    "modbus_channel_type": "discrete_input",
                },
            ],
        },
    }

    def setup(self) -> None:
        """
        Select task configuration based on matrix parameter and setup task.

        Matrix Parameters:
            task (str): Task type selector. Valid values:
                - "input_register": Read input registers (FC 4, addresses 0-1)
                - "holding_register": Read holding registers (FC 3, addresses 0-1)
                - "discrete_input": Read discrete inputs (FC 2, addresses 0-1)
                - "coil": Read coils (FC 1, addresses 0-1)

        Example JSON configurations:
            {"case": "driver/modbus_read", "task": "input_register"}
            {"case": "driver/modbus_read", "task": "holding_register"}
            {"case": "driver/modbus_read", "task": "discrete_input"}
            {"case": "driver/modbus_read", "task": "coil"}
        """
        # Select task configuration
        task_type = self.params.get("task", "mixed")
        if task_type not in self.TASK_CONFIGS:
            self.fail(f"Unknown task_type: {task_type}")
            return

        config = self.TASK_CONFIGS[task_type]
        self.TASK_NAME = config["task_name"]
        self.TASK_KEY = config["task_key"]
        self.CHANNELS = config["channels"]

        super().setup()

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
        # Create appropriate channel types based on per-channel type
        task_channels = []
        for ch, meta in zip(channels, channel_metadata):
            channel_type = meta["modbus_channel_type"]

            if channel_type == "input_register":
                task_channels.append(
                    modbus.InputRegisterChan(
                        channel=ch.key,
                        address=meta["address"],
                        data_type=meta["modbus_data_type"],
                    )
                )
            elif channel_type == "holding_register":
                task_channels.append(
                    modbus.HoldingRegisterInputChan(
                        channel=ch.key,
                        address=meta["address"],
                        data_type=meta["modbus_data_type"],
                    )
                )
            elif channel_type == "discrete_input":
                task_channels.append(
                    modbus.DiscreteInputChan(
                        channel=ch.key,
                        address=meta["address"],
                    )
                )
            elif channel_type == "coil":
                task_channels.append(
                    modbus.CoilInputChan(
                        channel=ch.key,
                        address=meta["address"],
                    )
                )
            else:
                raise ValueError(f"Unknown channel type: {channel_type}")

        return modbus.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=task_channels,
        )
