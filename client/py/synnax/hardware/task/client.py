#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import warnings
from typing import overload, Protocol
from pydantic import ValidationError

from alamos import NOOP, Instrumentation
from freighter import UnaryClient, Payload, send_required, Empty
from uuid import uuid4

from synnax import UnexpectedError
from synnax.hardware.task.payload import TaskPayload, TaskState
from synnax.framer import Client as FrameClient
from synnax.telem import TimeStamp, TimeSpan
from synnax.util.normalize import normalize, override, check_for_none
from synnax.hardware.rack import Rack, Client as RackClient
from synnax.exceptions import ConfigurationError


class _CreateRequest(Payload):
    tasks: list[TaskPayload]


_CreateResponse = _CreateRequest


class _DeleteRequest(Payload):
    keys: list[int]


class _RetrieveRequest(Payload):
    rack: int | None = None
    keys: list[int] | None = None
    names: list[str] | None = None
    types: list[str] | None = None
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
    _frame_client: FrameClient | None = None

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
        self._frame_client = _frame_client

    def to_payload(self) -> TaskPayload:
        return TaskPayload(
            key=self.key,
            name=self.name,
            type=self.type,
            config=self.config,
        )

    def set_internal(self, task: Task):
        self.key = task.key
        self.name = task.name
        self.type = task.type
        self.config = task.config
        self.snapshot = task.snapshot
        self._frame_client = task._frame_client

    def execute_command(self, type_: str, args: dict | None = None) -> str:
        """Executes a command on the task and returns the unique key assigned to the
        command.

        :param type_: The type of command to execute.
        :param args: The arguments to pass to the command.
        :return: The unique key assigned to the command.
        """
        w = self._frame_client.open_writer(TimeStamp.now(), _TASK_CMD_CHANNEL)
        key = str(uuid4())
        w.write(
            _TASK_CMD_CHANNEL,
            [{"task": self.key, "type": type_, "key": key, "args": args}],
        )
        w.close()
        return str(key)

    def execute_command_sync(
        self,
        type_: str,
        args: dict | None = None,
        timeout: float | TimeSpan = 5,
    ) -> str:
        """Executes a command on the task and waits for the driver to acknowledge the
        command with a state.

        :param type_: The type of command to execute.
        :param args: The arguments to pass to the command.
        :param timeout: The maximum time to wait for the driver to acknowledge the
        command before a timeout occurs.
        """
        with self._frame_client.open_streamer([_TASK_STATE_CHANNEL]) as s:
            key = self.execute_command(type_, args)
            while True:
                print(TimeSpan.from_seconds(timeout).seconds)
                frame = s.read(TimeSpan.from_seconds(timeout).seconds)
                if frame is None:
                    raise TimeoutError(
                        f"timed out waiting for driver to acknowledge {type_} command"
                    )
                elif _TASK_STATE_CHANNEL not in frame:
                    warnings.warn("task - unexpected missing state in frame")
                    continue
                try:
                    state = TaskState.parse_obj(frame[_TASK_STATE_CHANNEL][0])
                    print(state)
                    if state.key == key:
                        return state
                except ValidationError as e:
                    raise UnexpectedError(
                        f"""
                    Received invalid task state from driver.
                    """
                    ) from e


class MetaTask(Protocol):
    key: int

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
        if tasks is None:
            tasks = [TaskPayload(key=key, name=name, type=type, config=config)]
        elif isinstance(tasks, Task):
            tasks = [tasks.to_payload()]
        else:
            is_single = False
            tasks = [t.to_payload() for t in tasks]
        for pld in tasks:
            self.maybe_assign_def_rack(pld, rack)
        req = _CreateRequest(tasks=tasks)
        res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
        st = self.sugar(res.tasks)
        return st[0] if is_single else st

    def maybe_assign_def_rack(self, pld: TaskPayload, rack: int = 0) -> Rack:
        if self._default_rack is None:
            # Hardcoded as this value for now. Will be changed once we have multi-rack
            # systems
            self._default_rack = self._racks.retrieve(name="sy_node_1_rack")
        if pld is not None and pld.key == 0:
            if rack == 0:
                rack = self._default_rack.key
            pld.key = (rack << 32) + 0
        return pld

    def configure(self, task: MetaTask, timeout: float = 5) -> MetaTask:
        with self._frame_client.open_streamer([_TASK_STATE_CHANNEL]) as streamer:
            pld = self.maybe_assign_def_rack(task.to_payload())
            req = _CreateRequest(tasks=[pld])
            res = send_required(self._client, _CREATE_ENDPOINT, req, _CreateResponse)
            task.set_internal(self.sugar(res.tasks)[0])
            while True:
                frame = streamer.read(timeout)
                if frame is None:
                    raise TimeoutError(
                        "task - timeout waiting for driver to "
                        "acknowledge configuration"
                    )
                elif (
                    _TASK_STATE_CHANNEL not in frame
                    or len(frame[_TASK_STATE_CHANNEL]) == 0
                ):
                    warnings.warn("task - unexpected missing state in frame")
                    continue
                state = frame["sy_task_state"][0]
                if int(state["task"]) != task.key:
                    continue
                variant = state["variant"]
                if variant == "success":
                    break
                if variant == "error":
                    raise ConfigurationError(state["details"]["message"])
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
        type: str | None = None,
    ) -> Task:
        ...

    @overload
    def retrieve(
        self,
        names: list[str] | None = None,
        keys: list[int] | None = None,
        types: list[str] | None = None,
    ) -> list[Task]:
        ...

    def retrieve(
        self,
        key: int | None = None,
        name: str | None = None,
        type: str | None = None,
        names: list[str] | None = None,
        keys: list[int] | None = None,
        types: list[str] | None = None,
    ) -> list[Task] | Task:
        is_single = check_for_none(names, keys, types)
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(
                keys=override(key, keys),
                names=override(name, names),
                types=override(type, types),
            ),
            _RetrieveResponse,
        )
        sug = self.sugar(res.tasks)
        return sug[0] if is_single else sug

    def sugar(self, tasks: list[Payload]):
        return [
            Task(
                **t.dict(),
                _frame_client=self._frame_client,
            )
            for t in tasks
        ]
