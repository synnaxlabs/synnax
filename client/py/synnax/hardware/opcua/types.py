#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from contextlib import contextmanager

from pydantic import BaseModel, conint

from synnax.telem import TimeSpan, CrudeRate
from synnax.hardware.task import MetaTask, TaskPayload, Task


class Channel:
    key: str
    channel: int
    nodeId: str
    enabled: bool = True
    use_as_index: bool = False


class ReadConfig(BaseModel):
    sample_rate: conint(g=0, le=50000)
    stream_rate: conint(g=0, le=50000)
    channels: list[Channel]
    array_mode: bool
    array_size: conint(g=0)
    data_saving: bool


class ReadTask(MetaTask):
    TYPE = "opc_read"
    config: ReadConfig
    _internal: Task

    def __init__(
        self,
        internal: Task | None = None,
        *,
        device: str = "",
        name: str = "",
        sample_rate: CrudeRate = 1000,
        stream_rate: CrudeRate = 1000,
        data_saving: bool = False,
        array_mode: bool = False,
        array_size: int = 0,
        channels: list[Channel] = None,
    ):
        if internal is not None:
            self._internal = internal
            self.config = ReadConfig.parse_obj(json.loads(internal.config))
            return
        self._internal = Task(name=name, type=self.TYPE)
        self.config = ReadConfig(
            device=device,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=data_saving,
            array_mode=array_mode,
            array_size=array_size,
            channels=channels,
        )

    def to_payload(self) -> TaskPayload:
        pld = self._internal.to_payload()
        pld.config = json.dumps(self.config.dict())
        return pld

    def set_internal(self, task: Task):
        self._internal = task

    def start(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("start", timeout=timeout)

    def stop(self, timeout: float | TimeSpan = 0):
        self._internal.execute_command_sync("stop", timeout=timeout)

    @contextmanager
    def read(self, timeout: float | TimeSpan = 0):
        self.start(timeout)
        try:
            yield
        finally:
            self.stop(timeout)
