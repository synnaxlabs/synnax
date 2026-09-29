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
from synnax.tcp.types_gen import (
    Properties,
    ReadConfig,
    WriteConfig,
)
from synnax.telem import CrudeRate, CrudeTimeSpan, Rate, TimeSpan

MAKE = "TCP"


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task that decodes TCP messages described by a library into
    channels.

    :param device: The key of the TCP device to read from.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to decode and the channels each field writes to.
    :param raw: The virtual bytes channel every received frame is written to. Zero
        when the task does not stream raw frames.
    :param rate: The rate at which the task sends the query of each polled
        message.
    :param timeout: How long the task waits for a reply before it counts a miss.
    :param framing: How the byte stream is split into frames. Defaults to
        newline-delimited frames.
    :param data_saving_disabled: Whether to only stream data instead of saving it.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "tcp_read"
    config: ReadConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        library: library.Key | None = None,
        messages: list[bus.ReadMessage] | None = None,
        raw: int = 0,
        rate: CrudeRate = 1,
        timeout: CrudeTimeSpan = TimeSpan.SECOND,
        framing: bus.Framing | None = None,
        data_saving_disabled: bool = False,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        if library is None:
            raise ValueError("library is required")
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            library=library,
            messages=messages if messages is not None else [],
            raw=raw,
            rate=Rate(rate),
            timeout=TimeSpan(timeout),
            framing=framing if framing is not None else bus.DelimiterFraming(),
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
        )

    def update_device_properties(self, device_client: device.Client) -> None:
        """Leaves the device unchanged: the task config holds the channel mapping."""
        return None


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task that encodes command channel values into TCP messages
    described by a library.

    :param device: The key of the TCP device to write to.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to send and the command channel of each field.
    :param framing: How the byte stream is split into frames. Defaults to
        newline-delimited frames.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "tcp_write"
    config: WriteConfig
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        library: library.Key | None = None,
        messages: list[bus.WriteMessage] | None = None,
        framing: bus.Framing | None = None,
        auto_start: bool = False,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        if library is None:
            raise ValueError("library is required")
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            library=library,
            messages=messages if messages is not None else [],
            framing=framing if framing is not None else bus.DelimiterFraming(),
            auto_start=auto_start,
        )

    def update_device_properties(self, device_client: device.Client) -> None:
        """Leaves the device unchanged: the task config holds the channel mapping."""
        return None


class Device(device.Device):
    """A TCP device: a device the Driver reaches as a TCP client.

    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Generated when empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        host: str,
        port: int,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        props = Properties(
            host=host,
            port=port,
        )
        super().__init__(
            key=key or str(uuid4()),
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model="TCP server",
            configured=configured,
            properties=props.model_dump(),
        )
