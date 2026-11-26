#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import override

from synnax import device, icd, task
from synnax.kafka.types_gen import (
    ReadConfig,
    ReadTopic,
    StartOffset,
    WriteConfig,
    WriteTarget,
)


class ReadTask(
    task.StarterStopperMixin, task.JSONConfigMixin[ReadConfig], task.Protocol
):
    """A read task that consumes topics of a Kafka cluster and writes the fields of
    the ICD messages in their records to Synnax channels.

    :param device: The key of the cluster device to consume from.
    :param name: A human-readable name for the task.
    :param data_saving_disabled: Whether to only stream data for real-time consumption
        instead of saving it permanently within Synnax.
    :param auto_start: Whether to start the task automatically.
    :param group: The consumer group the task joins. Empty derives the group from the
        task key.
    :param start_offset: Where the group starts when it has no committed offset.
    :param topics: The topics to consume.
    """

    TYPE = "kafka_read"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        data_saving_disabled: bool = False,
        auto_start: bool = False,
        group: str = "",
        start_offset: StartOffset = "latest",
        topics: list[ReadTopic] | None = None,
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
            group=group,
            start_offset=start_offset,
            topics=topics if topics is not None else [],
        )


class WriteTask(
    task.StarterStopperMixin, task.JSONConfigMixin[WriteConfig], task.Protocol
):
    """A write task that sends a record to a Kafka topic each time a value is written
    to a command channel of a target.

    :param device: The key of the cluster device to produce to.
    :param name: A human-readable name for the task.
    :param auto_start: Whether to start the task automatically.
    :param targets: The topics and ICD messages to produce.
    """

    TYPE = "kafka_write"
    _internal: task.Task

    def __init__(
        self,
        internal: task.Task | None = None,
        *,
        device: device.Key = "",
        name: str = "",
        auto_start: bool = False,
        targets: list[WriteTarget] | None = None,
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

    @override
    def update_device_properties(self, device_client: device.Client) -> device.Device:
        """Sync the command channel of each field of each enabled target to device
        properties, keyed by "topic#record_key#field", so each task on the device that
        sends to the same topic and record key shares the channel."""
        dev = device_client.retrieve(key=self.config.device)
        props = dict(dev.properties)
        write = device.ensure_object(props, "write")
        for t in self.config.targets:
            if t.disabled:
                continue
            for binding in t.message.bindings:
                if isinstance(binding, icd.ChannelWriteBinding):
                    write[f"{t.topic}#{t.record_key}#{binding.field}"] = binding.channel
        dev.properties = props
        device_client.dispatch(
            dev.key, device.set(device.SetPayload(path=["properties"], value=props))
        )
        return dev
