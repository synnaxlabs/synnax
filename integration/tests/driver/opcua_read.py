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
from synnax.hardware import opcua

from driver.devices import Simulator
from driver.driver import ChannelConfig
from tests.driver.task import TaskCase


class TaskTypeConfig(TypedDict, total=False):
    """
    Configuration for a specific OPC UA task type.

    Attributes:
        task_name: Human-readable name for the task
        task_key: Unique identifier for the task (used for index channel naming)
        channels: List of channel configurations with OPC UA-specific metadata
        array_mode: Whether to read array data (True) or scalar values (False)
        array_size: Number of elements per array when array_mode is True
    """

    task_name: str
    task_key: str
    channels: list[ChannelConfig]
    array_mode: bool
    array_size: int


class OpcuaRead(TaskCase):
    """
    OPC UA read task lifecycle test.

    This test configures and validates OPC UA read tasks with different channel types.
    The task type is selected via the 'task' matrix parameter.

    Supported task types:
    - "float": Standard float32 channels
    - "bool": Boolean channels (UINT8)
    - "array": Array mode float32 channels
    - "mixed": Scalar mode with both float32 and boolean channels
    """

    # Simulator configuration
    simulator = Simulator.OPCUA

    # OPC specific params
    array_mode: bool = False
    array_size: int = 5

    # Task configurations for different types
    TASK_CONFIGS: dict[str, TaskTypeConfig] = {
        "float": {
            "task_name": "OPC UA Py - Read Float",
            "task_key": "opcua_read_float",
            "channels": [
                {
                    "name": "my_float_0",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=8",
                    "opcua_data_type": "float32",
                },
                {
                    "name": "my_float_1",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=9",
                    "opcua_data_type": "float32",
                },
            ],
        },
        "bool": {
            "task_name": "OPC UA Py - Read Bool",
            "task_key": "opcua_read_bool",
            "channels": [
                {
                    "name": "my_bool_0",
                    "data_type": sy.DataType.UINT8,
                    "node_id": "NS=2;I=13",
                    "opcua_data_type": "bool",
                },
                {
                    "name": "my_bool_1",
                    "data_type": sy.DataType.UINT8,
                    "node_id": "NS=2;I=14",
                    "opcua_data_type": "bool",
                },
            ],
        },
        "array": {
            "task_name": "OPC UA Py - Read Array",
            "task_key": "opcua_read_array",
            "array_mode": True,
            "array_size": 5,
            "channels": [
                {
                    "name": "my_array_0",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=2",
                    "opcua_data_type": "float32",
                },
                {
                    "name": "my_array_1",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=3",
                    "opcua_data_type": "float32",
                },
            ],
        },
        "mixed": {
            "task_name": "OPC UA Py - Read Mixed",
            "task_key": "opcua_read_mixed",
            "channels": [
                {
                    "name": "my_float_0",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=8",
                    "opcua_data_type": "float32",
                },
                {
                    "name": "my_float_1",
                    "data_type": sy.DataType.FLOAT32,
                    "node_id": "NS=2;I=9",
                    "opcua_data_type": "float32",
                },
                {
                    "name": "my_bool_0",
                    "data_type": sy.DataType.UINT8,
                    "node_id": "NS=2;I=13",
                    "opcua_data_type": "bool",
                },
                {
                    "name": "my_bool_1",
                    "data_type": sy.DataType.UINT8,
                    "node_id": "NS=2;I=14",
                    "opcua_data_type": "bool",
                },
            ],
        },
    }

    def setup(self) -> None:
        """
        Select task configuration based on matrix parameter and setup task.

        Matrix Parameters:
            task (str): Task type selector. Valid values:
                - "float": Read scalar float32 channels (NodeIds NS=2;I=8, NS=2;I=9)
                - "bool": Read scalar boolean channels (NodeIds NS=2;I=13, NS=2;I=14)
                - "array": Read array float32 channels (NodeIds NS=2;I=2, NS=2;I=3)
                - "mixed": Read scalar mixed channels (floats + bools)

        Example JSON configurations:
            {"case": "driver/opcua_read", "task": "float"}
            {"case": "driver/opcua_read", "task": "bool"}
            {"case": "driver/opcua_read", "task": "array"}
            {"case": "driver/opcua_read", "task": "mixed"}
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
        self.array_mode = config.get("array_mode", False)
        self.array_size = config.get("array_size", 5)

        super().setup()

    def create_task(
        self,
        device: sy.Device,
        channels: list[sy.Channel],
        channel_metadata: list[ChannelConfig],
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> opcua.ReadTask:
        """
        Create an OPC UA read task from channel metadata.

        Args:
            device: Synnax device to read from
            channels: Created Synnax channels
            channel_metadata: List of dicts with protocol-specific config
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured OPC UA read task
        """
        task_channels = [
            opcua.ReadChannel(
                channel=ch.key,
                node_id=meta["node_id"],
                data_type=meta["opcua_data_type"],
            )
            for ch, meta in zip(channels, channel_metadata)
        ]

        return opcua.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            array_mode=self.array_mode,
            array_size=self.array_size,
            channels=task_channels,
        )
