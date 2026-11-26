#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax import device, icd, task
from synnax.udp.types_gen import ReadConfig, WriteConfig


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task that reads the datagrams of a UDP socket as ICD messages and writes
    their fields to Synnax channels.

    :param device: The key of the UDP device to read from.
    :param name: A human-readable name for the task.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically.
    :param entries: The devices on the socket and the messages each one sends.
    """

    TYPE = "udp_read"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        entries: list[icd.ReadEntry] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            data_saving_disabled=data_saving_disabled,
            auto_start=auto_start,
            entries=entries if entries is not None else [],
        )


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task that sends ICD messages as UDP datagrams from the values written to
    the command channels of its targets.

    :param device: The key of the UDP device to write to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically.
    :param targets: The messages to send to the remote.
    """

    TYPE = "udp_write"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        targets: list[icd.WriteTarget] | None = None,
    ) -> None:
        if internal is not None:
            self._internal = internal
            self.config = WriteConfig.model_validate(internal.config)
            return
        self._internal = task.Task(name=name, type=self.TYPE)
        self.config = WriteConfig(
            device=device,
            auto_start=auto_start,
            targets=targets if targets is not None else [],
        )
