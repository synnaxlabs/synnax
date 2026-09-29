#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

from synnax import bus, device, library, task
from synnax.telem import CrudeRate, CrudeTimeSpan, Rate, TimeSpan
from synnax.udp.types_gen import (
    Properties,
    ReadConfig,
    WriteConfig,
)

MAKE = "UDP"


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task that decodes UDP messages described by a library into
    channels.

    :param device: The key of the UDP device to read from.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to decode and the channels each field writes to.
    :param raw: The virtual bytes channel every received frame is written to. Zero
        when the task does not stream raw frames.
    :param rate: The rate at which the task sends the query of each polled
        message.
    :param timeout: How long the task waits for a reply before it counts a miss.
    :param data_saving_disabled: Whether to only stream data instead of saving it.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "udp_read"
    config: ReadConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        library: library.Key,
        messages: list[bus.ReadMessage] | None = None,
        raw: int = 0,
        rate: CrudeRate = 1,
        timeout: CrudeTimeSpan = TimeSpan.SECOND,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            library=library,
            messages=messages if messages is not None else [],
            raw=raw,
            rate=Rate(rate),
            timeout=TimeSpan(timeout),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
        )


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task that encodes command channel values into UDP messages
    described by a library.

    :param device: The key of the UDP device to write to.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to send and the command channel of each field.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "udp_write"
    config: WriteConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        library: library.Key,
        messages: list[bus.WriteMessage] | None = None,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            library=library,
            messages=messages if messages is not None else [],
            auto_start=auto_start,
        )


class Device(device.Device):
    """A UDP device: a local UDP port.

    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Generated when empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        port: int,
        remote_host: str = "",
        remote_port: int = 0,
        multicast_group: str = "",
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        props = Properties(
            port=port,
            remote_host=remote_host,
            remote_port=remote_port,
            multicast_group=multicast_group,
        )
        super().__init__(
            key=key or str(uuid4()),
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model="UDP socket",
            configured=configured,
            properties=props.model_dump(),
        )
