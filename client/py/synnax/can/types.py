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
from synnax.can.types_gen import (
    Backend,
    Properties,
    ReadConfig,
    WriteConfig,
)

MAKE = "CAN"


class ReadTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A read task that decodes CAN messages described by a library into
    channels.

    :param device: The key of the CAN device to read from.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to decode and the channels each field writes to.
    :param raw: The virtual bytes channel every received frame is written to. Zero
        when the task does not stream raw frames.
    :param data_saving_disabled: Whether to only stream data instead of saving it.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "can_read"
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
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
        )


class WriteTask(task.StarterStopperMixin, task.JSONConfigMixin, task.Protocol):
    """A write task that encodes command channel values into CAN messages
    described by a library.

    :param device: The key of the CAN device to write to.
    :param name: A human-readable name for the task.
    :param library: The key of the library that describes the messages.
    :param messages: The messages to send and the command channel of each field.
    :param auto_start: Whether to start the task when it is created.
    """

    TYPE = "can_write"
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
    """A CAN device: one channel on one CAN adapter.

    :param name: Human-readable name for the device.
    :param location: Physical location or description.
    :param rack: Rack key this device belongs to.
    :param key: Unique key for the device. Generated when empty.
    :param configured: Whether the device has been configured.
    """

    def __init__(
        self,
        *,
        backend: Backend = "socketcan",
        channel: str,
        bitrate: int = 500000,
        fd: bool = False,
        data_bitrate: int = 2000000,
        listen_only: bool = False,
        name: str = "",
        location: str = "",
        rack: int = 0,
        key: str = "",
        configured: bool = False,
    ):
        props = Properties(
            backend=backend,
            channel=channel,
            bitrate=bitrate,
            fd=fd,
            data_bitrate=data_bitrate,
            listen_only=listen_only,
        )
        super().__init__(
            key=key or str(uuid4()),
            location=location,
            rack=rack,
            name=name,
            make=MAKE,
            model=backend,
            configured=configured,
            properties=props.model_dump(),
        )
