#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings
from typing import overload, Protocol

from alamos import NOOP, Instrumentation
from freighter import UnaryClient, Payload, send_required, Empty
from uuid import uuid4
from synnax.hardware.task.payload import TaskPayload
from synnax.framer import Client as FrameClient
from synnax.telem import TimeStamp
from synnax.util.normalize import normalize, override, check_for_none
from synnax.hardware.rack import Rack, Client as RackClient


class _CreateRequest(Payload):
    tasks: list[TaskPayload]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[int]


class _RetrieveRequest(Payload):
    rack: int | None = None
    keys: list[int] | None = None
    names: list[str] | None = None
    include_state: bool = False


class _RetrieveResponse(Payload):
    tasks: list[TaskPayload] | None = None


_CREATE_ENDPOINT = "/hardware/task/create"
_DELETE_ENDPOINT = "/hardware/task/delete"
_RETRIEVE_ENDPOINT = "/hardware/task/retrieve"

_TASK_STATE_CHANNEL = "sy_task_state"
_TASK_CMD_CHANNEL = "sy_task_cmd"


class Task:
    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False
    __frame_client: FrameClient | None = None

    def __init__(
        self,
        *,
        key: int = 0,
        rack: int = 0,
        name: str = "",
        type: str = "",
        config: str = "",
        snapshot: bool = False,
        _frame_client: FrameClient | None = None,
    ):
        if key == 0:
            key = (rack << 32) + 0
        self.key = key
        self.name = name
        self.type = type
        self.config = config
        self.snapshot = snapshot
        self.__frame_client = _frame_client

    def to_payload(self) -> TaskPayload:
        return TaskPayload(
            key=self.key,
            name=self.name,
            type=self.type,
            config=self.config,
        )

    def execute_command(self, type_: str, args: dict | None = None) -> str:
        w = self.__frame_client.open_writer(TimeStamp.now(), _TASK_CMD_CHANNEL)
        key = str(uuid4())
        w.write(
            _TASK_CMD_CHANNEL,
            [{"task": self.key, "type": type_, "key": key, "args": args}],
        )
        w.close()
        return str(key)


class MetaTask(Protocol):
    def to_payload(self) -> TaskPayload:
        ...

    def set_internal(self, task: Task):
        ...


class Client:
    _client: UnaryClient
    _frame_client: FrameClient
    _default_rack: Rack | None
    _racks: RackClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        frame_client: FrameClient,
        rack_client: RackClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self._frame_client = frame_client
        self._racks = rack_client
        self._default_rack = None
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        key: int = 0,
        name: str = "",
        type: str = "",
        config: str = "",
        rack: int = 0,
    ):
        ...

    @overload
    def create(self, tasks: Task) -> Task:
        ...

    @overload
    def create(self, tasks: list[Task]) -> list[Task]:
        ...

    def create(
        self,
        tasks: Task | list[Task] | None = None,
        *,
        key: int = 0,
        name: str = "",
        type: str = "",
        config: str = "",
        rack: int = 0,
    ) -> Task | list[Task]:
        is_single = True
        if key == 0:
            # rack as first 32 bits, 0 as last 32 bits
            key = (rack << 32) + 0
        if tasks is None:
            tasks = [TaskPayload(key=key, name=name, type=type, config=config)]
        elif isinstance(tasks, Task):
            tasks = [tasks.to_payload()]
        else:
            is_single = False
            tasks = [t.to_payload() for t in tasks]
        req = _CreateRequest(tasks=tasks)
        res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
        st = self.__sugar(res.tasks)
        if is_single:
            return st[0]
        return st

    def get_default_rack(self) -> Rack:
        if self._default_rack is None:
            self._default_rack = self._racks.retrieve(names=["sy_node_1_rack"])[0]
        return self._default_rack

    def configure(self, task: MetaTask) -> MetaTask:
        with self._frame_client.open_streamer([_TASK_STATE_CHANNEL]) as streamer:
            pld = task.to_payload()
            if pld.key == 0:
                pld.key = (self.get_default_rack().key << 32) + 0
            req = _CreateRequest(tasks=[pld])
            res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
            task.set_internal(self.__sugar(res.tasks)[0])
            while True:
                frame = streamer.read(5)
                if frame is None:
                    break
                elif (
                    _TASK_STATE_CHANNEL not in frame
                    or len(frame[_TASK_STATE_CHANNEL]) == 0
                ):
                    warnings.warn("task - unexpected missing state in frame")
                    continue
                state = frame["sy_task_state"][0]
                if state["task"] != task.key:
                    continue
                variant = state["variant"]
                if variant == "success":
                    break
                if variant == "error":
                    raise Exception(state["details"]["message"])
        return task

    def delete(self, keys: int | list[int]):
        req = _DeleteRequest(keys=normalize(keys))
        send_required(self._client, _DELETE_ENDPOINT, req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: int | None = None,
        name: str | None = None,
    ) -> Task:
        ...

    @overload
    def retrieve(
        self,
        names: list[str] | None = None,
        keys: list[int] | None = None,
    ) -> list[Task]:
        ...

    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        names: list[str] | None = None,
        keys: list[int] | None = None,
    ) -> list[Task] | Task:
        is_single = check_for_none(names, keys)
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(keys=override(key, keys), names=override(name, names)),
            _RetrieveResponse,
        )
        sug = self.__sugar(res.tasks)
        return sug[0] if is_single else sug

    def __sugar(self, tasks: list[Payload]):
        return [
            Task(
                **t.dict(),
                _frame_client=self._frame_client,
            )
            for t in tasks
        ]
