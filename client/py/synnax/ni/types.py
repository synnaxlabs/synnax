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
from synnax.exceptions import ValidationError
from synnax.ni.types_gen import (
    AIChannel,
    AnalogReadConfig,
    AnalogWriteConfig,
    AOChannel,
    CIChannel,
    CounterReadConfig,
    DIChannel,
    DigitalReadConfig,
    DigitalWriteConfig,
    DOChannel,
)
from synnax.telem import CrudeRate, Rate

# Device identifiers - must match Console expectations
MAKE = "NI"


def _assign_channel_devices(
    channels: list[AIChannel] | list[CIChannel], device_key: device.Key
) -> None:
    """Assigns the task-level device to channels missing one; raises when neither the
    channel nor the task carries a device."""
    for i, channel in enumerate(channels):
        if len(channel.device) == 0:
            if len(device_key) == 0:
                raise ValidationError(
                    f"No device provided for channel {i + 1} in task and no device "
                    "provided directly to the task. Please provide a device for the "
                    "channel or set the device for the task."
                )
            channel.device = device_key


class AnalogReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for reading analog data from NI devices and writing them to a Synnax
    cluster. For detailed information on configuring and operating an analog read
    task, see https://docs.synnaxlabs.com/reference/driver/ni/analog-read-task

    :param device: The key of the Synnax NI device to read from. Used as the default
        for channels that do not carry their own device.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the NI device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The analog input channels to acquire data from (AIChannel
        variants).
    """

    TYPE = "ni_analog_read"
    config: AnalogReadConfig
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
        channels: list[AIChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = AnalogReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = AnalogReadConfig(
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)
        _assign_channel_devices(self.config.channels, device)


class AnalogWriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for writing analog output data to NI devices. For detailed information
    on configuring and operating an analog write task, see
    https://docs.synnaxlabs.com/reference/driver/ni/analog-write-task

    :param device: The key of the Synnax NI device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream state data for real-time
        consumption instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The analog output channels to write to (AOChannel variants).
    """

    TYPE = "ni_analog_write"
    config: AnalogWriteConfig
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
        channels: list[AOChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = AnalogWriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = AnalogWriteConfig(
            device=device,
            state_rate=Rate(state_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)


class CounterReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for reading counter data from NI devices and writing them to a Synnax
    cluster. For detailed information on configuring and operating a counter read
    task, see https://docs.synnaxlabs.com/reference/driver/ni/counter-read-task

    :param device: The key of the Synnax NI device to read from. Used as the default
        for channels that do not carry their own device.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the NI device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The counter input channels to acquire data from (CIChannel
        variants).
    """

    TYPE = "ni_counter_read"
    config: CounterReadConfig
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
        channels: list[CIChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = CounterReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = CounterReadConfig(
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)
        _assign_channel_devices(self.config.channels, device)


class DigitalReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for reading digital data from NI devices and writing them to a Synnax
    cluster. For detailed information on configuring and operating a digital read
    task, see https://docs.synnaxlabs.com/reference/driver/ni/digital-read-task

    :param device: The key of the Synnax NI device to read from.
    :param name: A human-readable name for the task.
    :param sample_rate: The rate at which to sample data from the NI device.
    :param stream_rate: The rate at which acquired data will be streamed to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The digital input channels to acquire data from.
    """

    TYPE = "ni_digital_read"
    config: DigitalReadConfig
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
        channels: list[DIChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = DigitalReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = DigitalReadConfig(
            device=device,
            sample_rate=Rate(sample_rate),
            stream_rate=Rate(stream_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)


class DigitalWriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A task for writing digital output data to NI devices. For detailed information
    on configuring and operating a digital write task, see
    https://docs.synnaxlabs.com/reference/driver/ni/digital-write-task

    :param device: The key of the Synnax NI device to write to.
    :param name: A human-readable name for the task.
    :param state_rate: The rate at which to write task channel states to the Synnax
        cluster.
    :param data_saving_disabled: Whether to only stream state data for real-time
        consumption instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically when it is created.
    :param channels: The digital output channels to write to.
    """

    TYPE = "ni_digital_write"
    config: DigitalWriteConfig
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
        channels: list[DOChannel] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = DigitalWriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = DigitalWriteConfig(
            device=device,
            state_rate=Rate(state_rate),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            channels=channels if channels is not None else [],
        )
        task.assign_keys(self.config.channels)


class Device(device.Device):
    """A National Instruments device.

    :param identifier: Channel name prefix for all channels on this device.
    :param name: Human-readable name for the device.
    :param model: Device model (e.g. "NI 9205", "NI 9263").
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Auto-generated if empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        identifier: str,
        name: str = "",
        model: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        if not key:
            key = str(uuid4())
        super().__init__(
            key=key,
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=model,
            configured=configured,
            properties={"identifier": identifier},
        )
