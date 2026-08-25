#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from synnax import device, task
from synnax.modbus.types_gen import (
    ReadChannel,
    ReadConfig,
    WriteChannel,
    WriteConfig,
)
from synnax.telem import CrudeRate, Rate

# Device identifiers - must match Console expectations
MAKE = "Modbus"
MODEL = "Modbus"

# Device map keys keep the released type spellings, so channels created before the
# labels were renamed keep matching.
_READ_NAME_TYPES = {
    "coil": "coil_input",
    "discrete_input": "discrete_input",
    "holding_register": "holding_register_input",
    "input_register": "register_input",
}

_WRITE_NAME_TYPES = {
    "coil": "coil_output",
    "holding_register": "holding_register_output",
}


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task for sampling data from Modbus TCP devices and writing the data to a
    Synnax cluster. For detailed information on configuring and operating a Modbus read
    task, see https://docs.synnaxlabs.com/reference/driver/modbus/read-task

    :param device: The key of the Synnax Modbus device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the Modbus device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The input channels to acquire data from (ReadChannel variants:
        CoilReadChannel, DiscreteInputReadChannel,
        HoldingRegisterReadChannel, InputRegisterReadChannel).
    """

    TYPE = "modbus_read"
    config: ReadConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        sample_rate: CrudeRate = 10,
        stream_rate: CrudeRate = 5,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        channels: list[ReadChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

        The key format follows Console's readMapKey convention:
        "{type}-{address}" for fixed-density channels, "{type}-{address}-{dataType}"
        for register channels, with underscores replaced by hyphens.
        """
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties) if dev.properties is not None else {}
        if "read" not in props:
            props["read"] = {"index": 0, "channels": {}}
        for ch in self.config.channels:
            key = f"{_READ_NAME_TYPES[ch.type]}-{ch.address}"
            if hasattr(ch, "data_type"):
                key += f"-{ch.data_type}"
            key = key.replace("_", "-")
            props["read"]["channels"][key] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task for sending commands to Modbus TCP devices. For detailed
    information on configuring and operating a Modbus write task, see
    https://docs.synnaxlabs.com/reference/driver/modbus/write-task

    :param device: The key of the Synnax Modbus device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The output channels to write to (WriteChannel variants:
        CoilWriteChannel, HoldingRegisterWriteChannel).
    """

    TYPE = "modbus_write"
    config: WriteConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        channels: list[WriteChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration.

        The key format follows Console's writeMapKey convention: "{type}-{address}"
        with underscores replaced by hyphens. Write channels omit the dataType because
        the driver converts values based on register type.
        """
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties) if dev.properties is not None else {}
        if "write" not in props:
            props["write"] = {"channels": {}}
        for ch in self.config.channels:
            key = f"{_WRITE_NAME_TYPES[ch.type]}-{ch.address}".replace("_", "-")
            props["write"]["channels"][key] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class Device(device.Device):
    """A Modbus TCP server device.

    :param host: The IP address or hostname of the Modbus server.
    :param port: The TCP port number, typically 502.
    :param swap_bytes: Whether to swap byte order within 16-bit words.
    :param swap_words: Whether to swap word order for 32-bit and larger values.
    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        host: str,
        port: int,
        swap_bytes: bool = False,
        swap_words: bool = False,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        if not key:
            key = str(uuid4())
        props = {
            "connection": {
                "host": host,
                "port": port,
                "swap_bytes": swap_bytes,
                "swap_words": swap_words,
            },
            "read": {"index": 0, "channels": {}},
            "write": {"channels": {}},
        }
        super().__init__(
            key=key,
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=MODEL,
            configured=configured,
            properties=props,
        )
