#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel

import synnax as sy
from x.strings import random_name

CHANNEL_FIELDS = {"channel", "cmd_channel", "state_channel", "command_channel"}


class TaskTargets:
    """Creates the channels, devices, and statuses that a task config names, since
    the Core refuses a config that names one it does not hold."""

    def __init__(self, client: sy.Synnax) -> None:
        self._client = client
        self._rack = client.racks.create(name=random_name()).key
        self._channels: dict[sy.channel.Key, sy.channel.Key] = {}
        self._devices: dict[sy.device.Key, sy.device.Key] = {}

    def attach(self, config: BaseModel) -> None:
        """Points each channel and device field of config at a new channel or device,
        one for each distinct key, and creates each status that config names."""
        for name in type(config).model_fields:
            value = getattr(config, name)
            if isinstance(value, BaseModel):
                self.attach(value)
            elif isinstance(value, list):
                for item in value:
                    if isinstance(item, BaseModel):
                        self.attach(item)
            elif isinstance(value, bool) or value in (0, ""):
                continue
            elif name in CHANNEL_FIELDS and isinstance(value, int):
                setattr(config, name, self._channel(value))
            elif name == "device" and isinstance(value, str):
                setattr(config, name, self._device(value))
            elif name == "status" and isinstance(value, str):
                self._status(value)

    def _channel(self, key: sy.channel.Key) -> sy.channel.Key:
        if key not in self._channels:
            self._channels[key] = self._client.channels.create(
                name=random_name(), virtual=True, data_type=sy.DataType.FLOAT32
            ).key
        return self._channels[key]

    def _device(self, key: sy.device.Key) -> sy.device.Key:
        if key not in self._devices:
            self._devices[key] = self._client.devices.create(
                sy.Device(
                    key=f"{key}-{random_name()}",
                    name=key,
                    rack=self._rack,
                    location="test",
                    make="test",
                    model="test",
                )
            ).key
        return self._devices[key]

    def _status(self, key: sy.status.Key) -> None:
        self._client.statuses.set(
            sy.Status(key=key, name=key, variant=sy.status.VARIANT_INFO, message=key)
        )
