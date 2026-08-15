#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Final, Literal, TypeAlias, get_args
from uuid import uuid4

from synnax import device, task
from synnax.labjack.types_gen import (
    InputChannel,
    OutputChannel,
    ReadConfig,
    WriteConfig,
)
from synnax.telem import CrudeRate, Rate

# Device identifiers - must match Console expectations
MAKE = "LabJack"
# Supported models
T4: Final = "LJM_dtT4"
T7: Final = "LJM_dtT7"
T8: Final = "LJM_dtT8"
SUPPORTED_MODELS: TypeAlias = Literal["LJM_dtT4", "LJM_dtT7", "LJM_dtT8"]


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task for sampling data from LabJack devices and writing the data to a
    Synnax cluster. For detailed information on configuring and operating a LabJack
    read task, see https://docs.synnaxlabs.com/reference/driver/labjack/read-task

    :param device: The key of the Synnax LabJack device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the LabJack device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The input channels to acquire data from (InputChannel variants:
        InputChannelAI, InputChannelDI, InputChannelTc).
    """

    TYPE = "labjack_read"
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
        channels: list[InputChannel] | None = None,
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
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties) if dev.properties is not None else {}
        if "read" not in props:
            props["read"] = {"index": 0, "channels": {}}
        for ch in self.config.channels:
            # Map port location -> channel key for Console
            props["read"]["channels"][ch.port] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task for sending commands to LabJack devices. For detailed information
    on configuring and operating a LabJack write task, see
    https://docs.synnaxlabs.com/reference/driver/labjack/write-task

    :param device: The key of the Synnax LabJack device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream state data for real-time
        consumption instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The output channels to write to (OutputChannel variants:
        OutputChannelAO, OutputChannelDO).
    """

    TYPE = "labjack_write"
    config: WriteConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        state_rate: CrudeRate = 10,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        channels: list[OutputChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            state_rate=Rate(state_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties) if dev.properties is not None else {}
        if "write" not in props:
            props["write"] = {"channels": {}}
        for ch in self.config.channels:
            # Map port location -> state_channel key for Console
            props["write"]["channels"][ch.port] = ch.state_channel
        dev.properties = props
        return device_client.create(dev)


class Device(device.Device):
    """A LabJack device.

    :param model: LabJack model. Use the module constants T4, T7, or T8.
    :param identifier: Device identifier: serial number, IP address, or device name.
    :param connection_type: Connection method: "ANY", "USB", "TCP", "ETHERNET", or
        "WIFI".
    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        model: SUPPORTED_MODELS,
        identifier: str,
        connection_type: str = "ANY",
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        valid_models = get_args(SUPPORTED_MODELS)
        if model not in valid_models:
            raise ValueError(
                f"Invalid model '{model}'. Must be one of: {list(valid_models)}"
            )
        if not key:
            key = str(uuid4())
        props = {
            "connection": {
                "identifier": identifier,
                "connection_type": connection_type,
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
            model=model,
            configured=configured,
            properties=props,
        )
