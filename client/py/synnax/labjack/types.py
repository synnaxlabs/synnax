#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Sequence
from typing import Final, Literal, TypeAlias, get_args

from synnax import device, task
from synnax.labjack.types_gen import (
    ReadChannel,
    ReadConfig,
    WriteChannel,
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

# The device property groups that map each port to its channels, by channel type. The
# Console reads the same groups to reuse a port's channels in new tasks.
_READ_PORT_TYPES: Final = {"analog": "AI", "thermocouple": "AI", "digital": "DI"}
_WRITE_PORT_TYPES: Final = {"analog": "AO", "digital": "DO"}


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task for sampling data from LabJack devices and writing the data to a
    Synnax cluster. For detailed information on configuring and operating a LabJack
    read task, see https://docs.synnaxlabs.com/reference/hardware/labjack/read-task

    :param device: The key of the Synnax LabJack device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the LabJack device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster. For example, a sample rate of 100Hz and a stream rate of 25Hz will
        result in groups of 4 samples being streamed to the cluster every 40ms.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The input channels to acquire data from (ReadChannel variants:
        AnalogReadChannel, DigitalReadChannel, ThermocoupleReadChannel).
    """

    TYPE = "labjack_read"
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
        channels: Sequence[ReadChannel] | None = None,
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
            channels=list(channels) if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        for ch in self.config.channels:
            port_type = _READ_PORT_TYPES[ch.type]
            device.ensure_object(props, port_type, "channels")[ch.port] = ch.channel
        dev.properties = props
        return device_client.create(dev)


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task for sending commands to LabJack devices. For detailed information
    on configuring and operating a LabJack write task, see
    https://docs.synnaxlabs.com/reference/hardware/labjack/write-task

    :param device: The key of the Synnax LabJack device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream state data for real-time
        consumption instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The output channels to write to (WriteChannel variants:
        AnalogWriteChannel, DigitalWriteChannel).
    """

    TYPE = "labjack_write"
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
        channels: Sequence[WriteChannel] | None = None,
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
            channels=list(channels) if channels is not None else [],
        )
        task.assign_keys(self.config.channels)

    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Update device properties before task configuration."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        for ch in self.config.channels:
            channels = device.ensure_object(
                props, _WRITE_PORT_TYPES[ch.type], "channels"
            )
            channels[ch.port] = {"command": ch.cmd_channel, "state": ch.state_channel}
        dev.properties = props
        return device_client.create(dev)


class Device(device.Device):
    """A LabJack device. The Driver finds connected LabJacks on its own and creates
    them with the same key, so retrieve a found device instead of creating one.

    :param model: LabJack model. Use the module constants T4, T7, or T8.
    :param serial_number: The serial number of the device, which is also its key.
    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        model: SUPPORTED_MODELS,
        serial_number: int,
        name: str = "",
        location: str = "",
        rack: int = 0,
        configured: bool = False,
    ):
        valid_models = get_args(SUPPORTED_MODELS)
        if model not in valid_models:
            raise ValueError(
                f"Invalid model '{model}'. Must be one of: {list(valid_models)}"
            )
        super().__init__(
            key=str(serial_number),
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=model,
            configured=configured,
        )
