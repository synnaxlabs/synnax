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
from collections.abc import Generator, Iterable, Mapping, Sequence
from contextlib import contextmanager
from typing import (
    TYPE_CHECKING,
    Any,
    Literal,
    Protocol as BaseProtocol,
    TypeVar,
    overload,
    override,
)
from uuid import UUID, uuid4

from pydantic import BaseModel, JsonValue, ValidationError
from pydantic_core import to_jsonable_python

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax import device as device_, framer, rack as rack_
from synnax.resource import ID
from synnax.status import VARIANT_ERROR, VARIANT_LOADING, VARIANT_WARNING
from synnax.task.types_gen import (
    Command,
    Cursor,
    Key,
    Order,
    Payload,
    RetrieveRequest,
    RetrieveResponse,
    Status,
    Where,
    WhereInput,
    resource_id,
)
from synnax.telem import CrudeTimeSpan, TimeSpan
from synnax.util import request_where, returns_one, sole
from x import filter
from x.timing import Timer

# synnax.arc imports synnax.task at load time.
if TYPE_CHECKING:
    from synnax import arc


class _CreateRequest(BaseModel):
    tasks: list[Payload]


_CreateResponse = _CreateRequest


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _CopyRequest(BaseModel):
    key: Key
    name: str


class _CopyResponse(BaseModel):
    task: Payload


_CREATE_ENDPOINT = "/task/create"
_DELETE_ENDPOINT = "/task/delete"
_RETRIEVE_ENDPOINT = "/task/retrieve"
_COPY_ENDPOINT = "/task/copy"

_TASK_STATE_CHANNEL = "sy_status_set"
_TASK_CMD_CHANNEL = "sy_task_cmd"
# The Driver fails an operation that takes longer than 60 s. A wait past this limit
# lost its command, for example to a Driver restart.
_FINAL_STATUS_TIMEOUT = 2 * TimeSpan.MINUTE


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
    rack: rack_.Key
    name: str
    type: str
    config: Mapping[str, object]
    config_hash: str
    status: Status | None
    _cached_frame_client: framer.Client | None

    def __init__(
        self,
        *,
        key: Key | None = None,
        rack: rack_.Key = 0,
        name: str = "",
        type: str = "",
        config: Mapping[str, object] | None = None,
        config_hash: str = "",
        status: Status | None = None,
        internal: bool = False,
        _frame_client: framer.Client | None = None,
    ):
        self.key = uuid4() if key is None else key
        self.rack = rack
        self.name = name
        self.type = type
        self.config = config if config is not None else {}
        self.config_hash = config_hash
        self.internal = internal
        self.status = status
        self._cached_frame_client = _frame_client

    @property
    def _frame_client(self) -> framer.Client:
        if self._cached_frame_client is None:
            raise RuntimeError(
                "Cannot execute commands on a task that has not been created or "
                "retrieved from the cluster."
            )
        return self._cached_frame_client

    def to_payload(self) -> Payload:
        return Payload(
            key=self.key,
            rack=self.rack,
            name=self.name,
            type=self.type,
            config=to_jsonable_python(self.config),
        )

    def set_internal(self, task: Task) -> None:
        self.key = task.key
        self.rack = task.rack
        self.name = task.name
        self.type = task.type
        self.config = task.config
        self.config_hash = task.config_hash
        self._cached_frame_client = task._cached_frame_client

    @property
    def resource_id(self) -> ID:
        return resource_id(self.key)

    def update_device_properties(
        self, device_client: device_.Client
    ) -> device_.Device | None:
        """Update device properties before task configuration.

        Default implementation for base Task class does nothing and returns None.
        Tasks that need to update device properties (LabJack, Modbus, OPC UA)
        should override this method in their respective classes.

        :returns: None - base implementation performs no updates.
        """
        return None

    def execute_command(
        self, type_: str, *, args: Mapping[str, object] | None = None
    ) -> str:
        """Executes a command on the task and returns the unique key assigned to the
        command.

        :param type_: The type of command to execute.
        :param args: The arguments to pass to the command.
        :returns: The unique key assigned to the command.
        """
        w = self._frame_client.open_writer(channels=_TASK_CMD_CHANNEL)
        cmd = Command(
            task=self.key,
            type=type_,
            key=str(uuid4()),
            config_hash=self.config_hash,
            args=to_jsonable_python(args) if args is not None else {},
        )
        w.write(_TASK_CMD_CHANNEL, [cmd.model_dump(mode="json")])
        w.close()
        return cmd.key

    def execute_command_sync(
        self,
        type_: str,
        *,
        args: Mapping[str, object] | None = None,
        timeout: CrudeTimeSpan = 5,
    ) -> Status:
        """Executes a command on the task and waits for the final status that the
        driver sends for it. A loading status acknowledges the command, and the wait
        goes on until a status that is not loading.

        :param type_: The type of command to execute.
        :param args: The arguments to pass to the command.
        :param timeout: How long to wait for the driver to acknowledge the command.
        Once it does, the wait for the final status lasts two minutes or the timeout,
        whichever is longer.
        :raises TimeoutError: If the driver does not acknowledge the command or send
        its final status in time.
        :raises RuntimeError: If the rack of the task reports that its driver is down
        while the command waits.
        """
        span = TimeSpan.from_seconds(timeout)
        rack_id = rack_.resource_id(self.rack) if self.rack != 0 else None
        with self._frame_client.open_streamer([_TASK_STATE_CHANNEL]) as s:
            key = self.execute_command(type_, args=args)
            timer = Timer()
            acknowledged = False
            while True:
                # The channel carries every status in the cluster, so the wait is a
                # deadline: a frame answering some other command must not renew it.
                remaining = span - timer.elapsed()
                frame = s.read(timeout=remaining) if remaining > TimeSpan.ZERO else None
                if frame is None:
                    step = "finish" if acknowledged else "acknowledge"
                    raise TimeoutError(
                        f"timed out waiting for driver to {step} {type_} command"
                    )
                if _TASK_STATE_CHANNEL not in frame:
                    warnings.warn(
                        "task - unexpected missing state in frame", stacklevel=2
                    )
                    continue
                # A frame can carry several statuses, and only some of them answer
                # this command.
                for sample in frame[_TASK_STATE_CHANNEL]:
                    if rack_id is not None:
                        _raise_if_rack_down(sample, rack_id)
                    try:
                        status = Status.model_validate(sample)
                    except ValidationError:
                        # The channel carries every status in the cluster, and a status
                        # that no task owns can carry other details.
                        continue
                    if status.details is not None and status.details.cmd == key:
                        if status.variant != VARIANT_LOADING:
                            return status
                        if not acknowledged:
                            acknowledged = True
                            span = max(span, _FINAL_STATUS_TIMEOUT)
                            timer = Timer()


def _raise_if_rack_down(sample: object, rack_id: ID) -> None:
    """Raises a status of the rack as an error when it reports that the Driver of the
    rack is down. A down Driver never replies, so the status ends the wait.
    """
    try:
        rack_status = rack_.Status.model_validate(sample)
    except ValidationError:
        return
    # Only the Core writes warnings to rack statuses, when it stops hearing from the
    # Driver.
    if rack_status.owner != rack_id or rack_status.variant != VARIANT_WARNING:
        return
    raise _status_error(rack_status.message, rack_status.description)


def _raise_if_failed(status: Status) -> None:
    """Raises the final status of a command as an error when the driver failed it."""
    if status.variant == VARIANT_ERROR:
        raise _status_error(status.message, status.description)


def _status_error(message: str, description: str) -> RuntimeError:
    return RuntimeError(f"{message}: {description}" if description else message)


class Protocol(BaseProtocol):
    @property
    def key(self) -> Key: ...

    def to_payload(self) -> Payload: ...

    def set_internal(self, task: Task) -> None: ...

    def update_device_properties(
        self, device_client: device_.Client
    ) -> device_.Device | None:
        """Update device properties before task configuration.

        This method can be overridden by tasks that need to synchronize
        their configuration with device properties (e.g., Modbus, OPC UA, LabJack).
        The default implementation does nothing.

        :param device_client: Client for accessing device operations
        :returns: The updated device, or None if no update was performed.
        """
        return None


P = TypeVar("P", bound=Protocol)


class StarterStopperMixin:
    _internal: Task

    def start(self, *, timeout: CrudeTimeSpan = 5) -> None:
        """Starts the task and blocks until the driver sends the final status of the
        start.

        :param timeout: How long to wait for the driver to acknowledge the start. Once
            it does, the wait for the final status lasts two minutes or the timeout,
            whichever is longer.
        :raises TimeoutError: If the driver does not acknowledge the start or send its
            final status in time.
        :raises RuntimeError: If the driver fails to start the task, or if the driver
            goes down during the wait.
        """
        _raise_if_failed(self._internal.execute_command_sync("start", timeout=timeout))

    def stop(self, *, timeout: CrudeTimeSpan = 5) -> None:
        """Stops the task and blocks until the driver sends the final status of the
        stop.

        :param timeout: How long to wait for the driver to acknowledge the stop. Once
            it does, the wait for the final status lasts two minutes or the timeout,
            whichever is longer.
        :raises TimeoutError: If the driver does not acknowledge the stop or send its
            final status in time.
        :raises RuntimeError: If the driver of the task goes down during the wait. An
            error status does not raise, because the stop status carries the last error
            of the task.
        """
        self._internal.execute_command_sync("stop", timeout=timeout)

    @contextmanager
    def run(self, *, timeout: CrudeTimeSpan = 5) -> Generator[None, None, None]:
        """Context manager that starts the task before entering the block and stops the
        task after exiting the block. This is useful for ensuring that the task is
        properly stopped even if an exception occurs during execution.
        """
        self.start(timeout=timeout)
        try:
            yield
        finally:
            self.stop(timeout=timeout)


class JSONConfigMixin[C: BaseModel](Protocol):
    _internal: Task
    config: C

    @property
    def name(self) -> str:
        return self._internal.name

    @override
    @property
    def key(self) -> Key:
        """Implements the Protocol interface."""
        return self._internal.key

    @override
    def to_payload(self) -> Payload:
        """Implements the Protocol interface."""
        pld = self._internal.to_payload()
        pld.config = to_jsonable_python(self.config, by_alias=True, exclude_none=True)
        return pld

    @override
    def set_internal(self, task: Task) -> None:
        """Implements the Protocol interface."""
        self._internal = task


class Client:
    _client: UnaryClient
    _frame_client: framer.Client
    _default_rack: rack_.Rack | None
    _racks: rack_.Client
    _device_client: device_.Client
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        *,
        client: UnaryClient,
        frame_client: framer.Client,
        rack_client: rack_.Client,
        device_client: device_.Client,
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
        key: Key | None = None,
        name: str,
        type: str = "",
        config: Mapping[str, object] | BaseModel | None = None,
        rack: rack_.Key = 0,
    ) -> Task: ...

    @overload
    def create(self, tasks: Task) -> Task: ...

    @overload
    def create(self, tasks: list[Task]) -> list[Task]: ...

    def create(
        self,
        tasks: Task | list[Task] | None = None,
        *,
        key: Key | None = None,
        name: str = "",
        type: str = "",
        config: Mapping[str, object] | BaseModel | None = None,
        rack: rack_.Key = 0,
    ) -> Task | list[Task]:
        is_single = True
        json_config: dict[str, JsonValue] = (
            to_jsonable_python(config, by_alias=True, exclude_none=True)
            if config is not None
            else {}
        )
        if tasks is None:
            if key is None:
                key = uuid4()
            payloads = [
                Payload(key=key, rack=rack, name=name, type=type, config=json_config)
            ]
        elif isinstance(tasks, Task):
            payloads = [tasks.to_payload()]
        else:
            is_single = False
            payloads = [t.to_payload() for t in tasks]
        for pld in payloads:
            self.maybe_assign_def_rack(pld, rack=rack)
        req = _CreateRequest(tasks=payloads)
        created = self._exec_create(req)
        sugared = self.sugar(created)
        return sugared[0] if is_single else sugared

    def _exec_create(self, req: _CreateRequest) -> list[Payload]:
        res = self._client.send("/task/create", req, _CreateResponse)
        return res.tasks

    def maybe_assign_def_rack(self, pld: Payload, *, rack: rack_.Key = 0) -> Payload:
        if pld.rack != 0:
            return pld
        if rack == 0:
            if self._default_rack is None:
                self._default_rack = self._racks.retrieve_embedded_rack()
            rack = self._default_rack.key
        pld.rack = rack
        return pld

    def configure(self, task: P) -> P:
        """Saves the task's configuration to the cluster. The config is deployed
        to the driver on the next start command. A task with no rack goes on the rack
        of the device that its config names, or on the embedded rack when it names no
        device.

        :param task: The task to save.
        :returns: The saved task with its key and rack populated.
        :raises NotFoundError: If the config names a device that does not exist.
        """
        task.update_device_properties(self._device_client)
        pld = task.to_payload()
        device = pld.config.get("device")
        if pld.rack == 0 and isinstance(device, str) and device:
            pld.rack = self._device_client.retrieve(key=device).rack
        pld = self.maybe_assign_def_rack(pld)
        req = _CreateRequest(tasks=[pld])
        tasks = self._exec_create(req)
        task.set_internal(self.sugar(tasks)[0])
        return task

    def delete(self, keys: Key | Sequence[Key]) -> None:
        req = _DeleteRequest(keys=[keys] if isinstance(keys, UUID) else list(keys))
        self._client.send("/task/delete", req, Empty)

    def rename(self, key: Key, name: str) -> None:
        """Renames a task.

        :param key: The key of the task to rename.
        :param name: The new name for the task.
        :raises NotFoundError: If no task has the given key.
        """
        self._client.send("/task/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, key: Key, *, include_status: bool = ...) -> Task: ...

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        type: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        internal: filter.BoolInput = ...,
        arcs: arc.Key | list[arc.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Task: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        type: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        internal: filter.BoolInput = ...,
        arcs: arc.Key | list[arc.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Task: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        type: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        internal: filter.BoolInput = ...,
        arcs: arc.Key | list[arc.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        types: list[str] | None = ...,
        search_term: str = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Task: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        type: filter.StringInput = ...,
        rack: filter.SetInput[rack_.Key] = ...,
        internal: filter.BoolInput = ...,
        arcs: arc.Key | list[arc.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        types: list[str] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        include_status: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Task]: ...

    def retrieve(
        self,
        key: filter.SetInput[Key] | None = None,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        types: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> list[Task] | Task:
        """Retrieves tasks from the Core. With no arguments, retrieves every task.

        Each filter field takes a value, a list of values, or an operator, and a task
        must match every field. A plain ``key``, by position or by keyword, or a plain
        ``name`` returns one task.

        :param key: Keeps tasks with this key.
        :param name: Keeps tasks with this name.
        :param type: Keeps tasks of this type.
        :param rack: Keeps tasks on this rack.
        :param internal: Keeps only internal tasks when true, or only user tasks when
            false.
        :param arcs: Keeps the tasks that run any of these Arcs.
        :param and_: Keeps tasks that match every filter in the list.
        :param or_: Keeps tasks that match any filter in the list.
        :param not_: Keeps tasks that do not match the filter.
        :param where: A filter tree that the tasks must also match.
        :param keys: Keeps tasks with one of these keys.
        :param names: Keeps tasks with one of these names.
        :param types: Keeps tasks of one of these types.
        :param search_term: A fuzzy search term matched against task names.
        :param limit: The maximum number of tasks to retrieve.
        :param order: The order of the tasks. Defaults to key order.
        :param after: Retrieves the tasks past this position in the order: the key of
            the last task of the page before, and its value of the order field.
        :param include_status: Whether to populate each task's status. Tasks come
            back with a null status when false.
        :param one: Returns one task for any filter.
        :returns: One task for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of tasks. The list leaves out keys that do not exist.
        :raises NotFoundError: If one task is wanted and none matches.
        :raises MultipleFoundError: If one task is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one task is wanted
            beside a limit or a cursor.
        """
        if key is not None:
            fields["key"] = key
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
            type=None if types is None else filter.String(in_=types),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
            include_status=include_status,
        )
        res = self._client.send(_RETRIEVE_ENDPOINT, req, RetrieveResponse)
        tasks = self.sugar(res.tasks)
        return sole("task", req, tasks) if single else tasks

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
                status=t.status,
                _frame_client=self._frame_client,
            )
            for t in tasks
        ]

    def list(self, rack: rack_.Key | None = None) -> list[Task]:
        """Lists all tasks on a rack. If no rack is specified, lists all tasks on the
        default rack. Excludes internal system tasks (scanner tasks and rack state).

        :param rack: The rack key to list tasks from. If None, uses the default rack.
        :returns: A list of all user-created tasks on the specified rack.
        """
        if rack is None:
            if self._default_rack is None:
                self._default_rack = self._racks.retrieve_embedded_rack()
            rack = self._default_rack.key

        return self.retrieve(rack=rack, internal=False)

    def copy(self, key: Key, name: str) -> Task:
        """Copies an existing task with a new name.

        :param key: The key of the task to copy.
        :param name: The name for the new task.
        :returns: The newly created task.
        """
        req = _CopyRequest(key=key, name=name)
        res = self._client.send(_COPY_ENDPOINT, req, _CopyResponse)
        return self.sugar([res.task])[0]
