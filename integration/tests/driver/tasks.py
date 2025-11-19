#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Task configurations and factories for driver integration tests.

This module provides:
- TaskCase: Dataclass for task test configurations
- Task factories: Protocol-specific task creation functions
- TASK_CASE: Registry of all available task configurations
- assemble_task: Helper to create channels and tasks from configuration
"""

from dataclasses import dataclass
from typing import Any, Callable

import synnax as sy
from synnax.hardware import modbus, opcua

from tests.driver.devices import Simulator

# Type alias for task factory functions
# Takes: (device, synnax_channels, channel_metadata_list, task_name)
# Returns: configured task
TaskFactory = Callable[[sy.Device, list[sy.Channel], list[dict], str], sy.Task]


@dataclass(frozen=True)
class TaskCase:
    """
    Configuration for a task test case test setup.

    Attributes:
        task_name: Name for the task instance
        channels: List of channel configs with protocol-specific metadata
        task_factory: Function to create protocol-specific task
        simulator: Optional simulator configuration (None for real hardware)

    Note:
        Index channel is auto-generated as "{task_name}_index"
    """

    task_name: str
    channels: list[dict[str, Any]]  # name, data_type, + protocol-specific keys
    task_factory: TaskFactory
    simulator: Simulator | None = None


def create_modbus_task(
    device: sy.Device,
    channels: list[sy.Channel],
    channel_metadata: list[dict],
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
            address=meta["address"],  # From server.py
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


def create_opcua_task(
    device: sy.Device,
    channels: list[sy.Channel],
    channel_metadata: list[dict],
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
            node_id=meta["node_id"],  # From server.py
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
        channels=task_channels,
    )


def create_opcua_array_task(
    device: sy.Device,
    channels: list[sy.Channel],
    channel_metadata: list[dict],
    task_name: str,
    sample_rate: sy.Rate,
    stream_rate: sy.Rate,
) -> opcua.ReadTask:
    """
    Create an OPC UA array read task from channel metadata.

    Args:
        device: Synnax device to read from
        channels: Created Synnax channels
        channel_metadata: List of dicts with protocol-specific config
        task_name: Name for the task
        sample_rate: Sampling rate for the task
        stream_rate: Streaming rate for the task (unused in array mode)

    Returns:
        Configured OPC UA array read task
    """
    task_channels = [
        opcua.ReadChannel(
            channel=ch.key,
            node_id=meta["node_id"],  # From server.py
            data_type=meta["opcua_data_type"],
        )
        for ch, meta in zip(channels, channel_metadata)
    ]

    return opcua.ReadTask(
        name=task_name,
        device=device.key,
        sample_rate=sample_rate,
        array_mode=True,
        array_size=5,  # Matches server ARRAY_SIZE
        data_saving=True,
        channels=task_channels,
    )


TASK_CASE: dict[str, TaskCase] = {
    "modbus": TaskCase(
        # See client/py/examples/modbus/server.py
        task_name="Modbus Py - Read Task",
        channels=[
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
        ],
        task_factory=create_modbus_task,
        simulator=Simulator.MODBUS,
    ),
    "opcua": TaskCase(
        # See client/py/examples/opcua/server.py
        task_name="OPC UA Py - Read Task",
        channels=[
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
        task_factory=create_opcua_task,
        simulator=Simulator.OPCUA,
    ),
    "opcua_bool": TaskCase(
        # See client/py/examples/opcua/read_task_boolean.py
        task_name="OPC UA Py - Read Task (Boolean)",
        channels=[
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
        task_factory=create_opcua_task,
        simulator=Simulator.OPCUA,
    ),
    "opcua_array": TaskCase(
        # See client/py/examples/opcua/read_task_array.py
        task_name="OPC UA Py - Read Task (Array)",
        channels=[
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
        task_factory=create_opcua_array_task,
        simulator=Simulator.OPCUA,
    ),
}


def assemble_task(
    client: sy.Synnax,
    device_name: str,
    config: TaskCase,
    task: str,
    sample_rate: sy.Rate,
    stream_rate: sy.Rate,
) -> tuple[sy.Task, list[str]]:
    """
    Create Synnax channels and task from configuration.

    Args:
        client: Synnax client instance
        device_name: Name of the hardware device to configure task for
        config: Task case configuration
        task: Task identifier (used for index channel naming)
        sample_rate: Sampling rate for the task
        stream_rate: Streaming rate for the task

    Returns:
        Tuple of (configured task, list of channel names)
    """
    # Retrieve the device
    device = client.hardware.devices.retrieve(name=device_name)
    # Auto-generate index channel name from task name
    index_channel_name = f"{task}_index"
    index_ch = client.channels.create(
        name=index_channel_name,
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )

    # Create data channels from config
    channels = []
    channel_names = []
    for ch_config in config.channels:
        ch = client.channels.create(
            name=ch_config["name"],
            index=index_ch.key,
            data_type=ch_config["data_type"],
            retrieve_if_name_exists=True,
        )
        channels.append(ch)
        channel_names.append(ch_config["name"])

    # Create task using protocol-specific factory
    task_instance = config.task_factory(
        device, channels, config.channels, config.task_name, sample_rate, stream_rate
    )

    return task_instance, channel_names
