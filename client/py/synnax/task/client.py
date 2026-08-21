#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import warnings
from collections.abc import Generator, Iterable
from contextlib import contextmanager
from typing import Any, overload
from typing import Protocol as BaseProtocol
from uuid import UUID, uuid4

from pydantic import BaseModel, Field, ValidationError

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.device import Client as DeviceClient
from synnax.device import Device
from synnax.framer import Client as FrameClient
from synnax.ontology.payload import ID
from synnax.rack import Client as RackClient
from synnax.rack import Rack
from synnax.task.types_gen import Command, Key, Payload, Status, ontology_id
from synnax.telem import TimeSpan, TimeStamp
from x.lists import check_for_none, normalize, override


class _CreateRequest(BaseModel):
    tasks: list[Payload]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _CopyRequest(BaseModel):
    key: Key
    name: str
    snapshot: bool


class _CopyResponse(BaseModel):
    task: Payload


class _RetrieveRequest(BaseModel):
    rack: int | None = None
    keys: list[Key] | None = None
    names: list[str] | None = None
    types: list[str] | None = None
    include_status: bool = False
    internal: bool | None = None
    snapshot: bool | None = None


class _RetrieveResponse(BaseModel):
    tasks: list[Payload] = Field(default_factory=list)


_CREATE_ENDPOINT = "/task/create"
_DELETE_ENDPOINT = "/task/delete"
_RETRIEVE_ENDPOINT = "/task/retrieve"
_COPY_ENDPOINT = "/task/copy"

_TASK_STATE_CHANNEL = "sy_status_set"
_TASK_CMD_CHANNEL = "sy_task_cmd"


class _Keyed(BaseProtocol):
    key: str


def assign_keys(records: Iterable[_Keyed]) -> None:
    """Assigns a fresh UUID key to every record whose key is empty.

    :param records: Task config records (e.g. channels, endpoints) with a string
        key field.
    """
    for record in records:
        if not record.key:
            record.key = str(uuid4())


class Task:
    key: Key
    rack: int = 0
    name: str = ""
    type: str = ""
    config: dict[str, Any] = {}
    config_hash: str = ""
    snapshot: bool = False
    status: Status | None = None
    _cached_frame_client: FrameClient | None = None

    def __init__(
        self,
        *,
        key: Key | str | None = None,
        rack: int = 0,
        name: str = "",
        type: str = "",
        config: dict[str, Any] | None = None,
        config_hash: str = "",
        snapshot: bool = False,
        status: Status | None = None,
        internal: bool = False,
        _frame_client: FrameClient | None = None,
    ):
        if key is None:
            key = uuid4()
        elif isinstance(key, str):
            key = UUID(key)
        self.key = key
        self.rack = rack
        self.name = name
        self.type = type
        self.config = config if config is not None else {}
        self.config_hash = config_hash
        self.internal = internal
        self.snapshot = snapshot
        self.status = status
        self._cached_frame_client = _frame_client

    @property
    def _frame_client(self) -> FrameClient:
        if self._cached_frame_client is None:
            raise RuntimeError(
                "Cannot execute commands on a task that has not been created or retrieved from the cluster."
            )
        return self._cached_frame_client

    def to_payload(self) -> Payload:
        return Payload(
            key=self.key,
            rack=self.rack,
            name=self.name,
            type=self.type,
            config=self.config,
        )

    def set_internal(self, task: Task) -> None:
        self.key = task.key
        self.rack = task.rack
        self.name = task.name
        self.type = task.type
        self.config = task.config
        self.config_hash = task.config_hash
        self.snapshot = task.snapshot
        self._cached_frame_client = task._cached_frame_client

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)

    def update_device_properties(self, device_client: DeviceClient) -> Device | None:
        """Update device properties before task configuration.

        Default implementation for base Task class does nothing and returns None.
        Tasks that need to update device properties (LabJack, Modbus, OPC UA)
        should override this method in their respective classes.

        Returns:
            None - base implementation performs no updates.
        """
        return None

    def execute_command(self, type_: str, args: dict[str, Any] | None = None) -> str:
        """Executes a command on the task and returns the unique key assigned to the
        command.

        :param type_: The type of command to execute.
        :param args: The arguments to pass to the command.
        :return: The unique key assigned to the command.
        """
        w = self._frame_client.open_writer(TimeStamp.now(), _TASK_CMD_CHANNEL)
        cmd = Command(
            task=self.key,
            type=type_,
            key=str(uuid4()),
            config_hash=self.config_hash,
            args=args or {},
        )
        w.write(_TASK_CMD_CHANNEL, [cmd.model_dump(mode="json")])
        w.close()
        return cmd.key

    def execute_command_sync(
        self,
        type_: str,
        args: dict[str, Any] | None = None,
        timeout: float | TimeSpan = 5,
    ) -> Status:
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
                frame = s.read(TimeSpan.from_seconds(timeout).seconds)
                if frame is None:
                    raise TimeoutError(
                        f"timed out waiting for driver to acknowledge {type_} command"
                    )
                elif _TASK_STATE_CHANNEL not in frame:
                    warnings.warn("task - unexpected missing state in frame")
                    continue
                # A frame can carry several statuses, and only some of them answer
                # this command.
                for sample in frame[_TASK_STATE_CHANNEL]:
                    try:
                        status = Status.model_validate(sample)
                    except ValidationError:
                        # The status channel carries statuses for all tasks and
                        # racks. Rack statuses have a different schema, so
                        # validation failures are expected and should be skipped.
                        continue
                    if (
                        status.details is not None
                        and status.details.cmd is not None
                        and status.details.cmd == key
                    ):
                        return status


class Protocol(BaseProtocol):
    @property
    def key(self) -> Key: ...

    def to_payload(self) -> Payload: ...

    def set_internal(self, task: Task) -> None: ...

    def update_device_properties(self, device_client: DeviceClient) -> Device | None:
        """Update device properties before task configuration.

        This method can be overridden by tasks that need to synchronize
        their configuration with device properties (e.g., Modbus, OPC UA, LabJack).
        The default implementation does nothing.

        Args:
            device_client: Client for accessing device operations

        Returns:
            The updated device, or None if no update was performed.
        """
        ...


class StarterStopperMixin:
    _internal: Task

    def start(self, timeout: float | TimeSpan = 5) -> None:
        """Starts the task and blocks until the Synnax cluster has acknowledged the
        command or the specified timeout has elapsed.

        :raises TimeoutError: If the timeout is reached before the Synnax cluster
            acknowledges the command.
        :raises Exception: If the Synnax cluster fails to start the task correctly.
        """
        self._internal.execute_command_sync("start", timeout=timeout)

    def stop(self, timeout: float | TimeSpan = 5) -> None:
        """Stops the task and blocks until the Synnax cluster has acknowledged the
        command or the specified timeout has elapsed.

        :raises TimeoutError: If the timeout is reached before the Synnax cluster
            acknowledges the command.
        :raises Exception: If the Synnax cluster fails to stop the task correctly.
        """
        self._internal.execute_command_sync("stop", timeout=timeout)

    @contextmanager
    def run(self, timeout: float | TimeSpan = 5) -> Generator[None, None, None]:
        """Context manager that starts the task before entering the block and stops the
        task after exiting the block. This is useful for ensuring that the task is
        properly stopped even if an exception occurs during execution.
        """
        self.start(timeout)
        try:
            yield
        finally:
            self.stop(timeout)


class JSONConfigMixin(Protocol):
    _internal: Task
    config: BaseModel

    @property
    def name(self) -> str:
        return self._internal.name

    @property
    def key(self) -> Key:
        """Implements TaskProtocol protocol"""
        return self._internal.key

    def to_payload(self) -> Payload:
        """Implements TaskProtocol protocol"""
        pld = self._internal.to_payload()
        pld.config = self.config.model_dump(by_alias=True, exclude_none=True)
        return pld

    def set_internal(self, task: Task) -> None:
        """Implements TaskProtocol protocol"""
        self._internal = task


class Client:
    _client: UnaryClient
    _frame_client: FrameClient
    _default_rack: Rack | None
    _racks: RackClient
    _device_client: DeviceClient | None
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        frame_client: FrameClient,
        rack_client: RackClient,
        device_client: DeviceClient | None = None,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self._frame_client = frame_client
        self._racks = rack_client
        self._device_client = device_client
        self._default_rack = None
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        key: Key | str | None = None,
        name: str = "",
        type: str = "",
        config: dict[str, Any] | BaseModel | None = None,
        rack: int = 0,
    ) -> Task: ...

    @overload
    def create(self, tasks: Task) -> Task: ...

    @overload
    def create(self, tasks: list[Task]) -> list[Task]: ...

    def create(
        self,
        tasks: Task | list[Task] | None = None,
        *,
        key: Key | str | None = None,
        name: str = "",
        type: str = "",
        config: dict[str, Any] | BaseModel | None = None,
        rack: int = 0,
    ) -> Task | list[Task]:
        is_single = True
        if config is None:
            config = dict()
        elif isinstance(config, BaseModel):
            config = config.model_dump(by_alias=True, exclude_none=True)
        if tasks is None:
            if key is None:
                key = uuid4()
            elif isinstance(key, str):
                key = UUID(key)
            payloads = [
                Payload(key=key, rack=rack, name=name, type=type, config=config or {})
            ]
        elif isinstance(tasks, Task):
            payloads = [tasks.to_payload()]
        else:
            is_single = False
            payloads = [t.to_payload() for t in tasks]
        for pld in payloads:
            self.maybe_assign_def_rack(pld, rack)
        req = _CreateRequest(tasks=payloads)
        created = self._exec_create(req)
        sugared = self.sugar(created)
        return sugared[0] if is_single else sugared

    def _exec_create(self, req: _CreateRequest) -> list[Payload]:
        res = self._client.send("/task/create", req, _CreateResponse)
        return res.tasks

    def maybe_assign_def_rack(self, pld: Payload, rack: int = 0) -> Payload:
        if pld is None or pld.rack != 0:
            return pld
        if rack == 0:
            if self._default_rack is None:
                self._default_rack = self._racks.retrieve_embedded_rack()
            rack = self._default_rack.key
        pld.rack = rack
        return pld

    def configure(self, task: Protocol, timeout: float = 5) -> Protocol:
        """Saves the task's configuration to the cluster. The config is deployed
        to the driver on the next start command.

        :param task: The task to save.
        :param timeout: Unused. Retained for backwards compatibility.
        :returns: The saved task with its key and rack populated.
        """
        # Call task-specific device property update (e.g., for Modbus, OPC UA, LabJack)
        if self._device_client is not None:
            task.update_device_properties(self._device_client)
        pld = self.maybe_assign_def_rack(task.to_payload())
        req = _CreateRequest(tasks=[pld])
        tasks = self._exec_create(req)
        task.set_internal(self.sugar(tasks)[0])
        return task

    def delete(self, keys: Key | str | list[Key | str]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/task/delete", req, Empty)

    @overload
    def retrieve(
        self,
        key: Key | str | None = None,
        name: str | None = None,
        type: str | None = None,
        *,
        include_status: bool = False,
    ) -> Task: ...

    @overload
    def retrieve(
        self,
        key: None = None,
        name: None = None,
        type: None = None,
        names: list[str] | None = None,
        keys: list[Key | str] | None = None,
        types: list[str] | None = None,
        *,
        include_status: bool = False,
    ) -> list[Task]: ...

    def retrieve(
        self,
        key: Key | str | None = None,
        name: str | None = None,
        type: str | None = None,
        names: list[str] | None = None,
        keys: list[Key | str] | None = None,
        types: list[str] | None = None,
        *,
        include_status: bool = False,
    ) -> list[Task] | Task:
        """Retrieves tasks matching the given filters.

        :param include_status: Whether to populate each task's status. Tasks come
            back with a null status when false.
        """
        is_single = check_for_none(names, keys, types)
        res = self._client.send(
            "/task/retrieve",
            _RetrieveRequest(
                keys=override(key, keys),
                names=override(name, names),
                types=override(type, types),
                include_status=include_status,
            ),
            _RetrieveResponse,
        )
        sug = self.sugar(res.tasks)

        # Warn if multiple tasks found when retrieving by name
        if is_single and name is not None and len(sug) > 1:
            task_keys = ", ".join(str(t.key) for t in sug)
            warnings.warn(
                f"Multiple tasks ({len(sug)}) found with name '{name}'. "
                f"Keys: [{task_keys}]. Returning the first task ({sug[0].key}).",
                UserWarning,
                stacklevel=2,
            )

        return sug[0] if is_single else sug

    def sugar(self, tasks: list[Payload]) -> list[Task]:
        return [
            Task(
                key=t.key,
                rack=t.rack,
                name=t.name,
                type=t.type,
                config=t.config,
                config_hash=t.config_hash,
                internal=t.internal,
                snapshot=t.snapshot,
                status=t.status,
                _frame_client=self._frame_client,
            )
            for t in tasks
        ]

    def list(self, rack: int | None = None) -> list[Task]:
        """Lists all tasks on a rack. If no rack is specified, lists all tasks on the
        default rack. Excludes internal system tasks (scanner tasks and rack state).

        :param rack: The rack key to list tasks from. If None, uses the default rack.
        :return: A list of all user-created tasks on the specified rack.
        """
        if rack is None:
            if self._default_rack is None:
                self._default_rack = self._racks.retrieve_embedded_rack()
            rack = self._default_rack.key

        res = self._client.send(
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(rack=rack, internal=False),
            _RetrieveResponse,
        )
        return self.sugar(res.tasks)

    def copy(
        self,
        key: Key | str,
        name: str,
    ) -> Task:
        """Copies an existing task with a new name.

        :param key: The key of the task to copy.
        :param name: The name for the new task.
        :return: The newly created task.
        """
        req = _CopyRequest(key=key, name=name, snapshot=False)
        res = self._client.send(_COPY_ENDPOINT, req, _CopyResponse)
        return self.sugar([res.task])[0]
