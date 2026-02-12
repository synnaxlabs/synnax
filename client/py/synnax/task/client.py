#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import json
import warnings
from contextlib import contextmanager
from typing import Protocol, overload
from uuid import uuid4

from alamos import NOOP, Instrumentation
from freighter import Empty, Payload, UnaryClient, send_required
from pydantic import BaseModel, Field, ValidationError, conint, field_validator

from synnax.device import Client as DeviceClient
from synnax.device import Device
from synnax.exceptions import ConfigurationError, UnexpectedError
from synnax.framer import Client as FrameClient
from synnax.rack import Client as RackClient
from synnax.rack import Rack
from synnax.status import VARIANT_ERROR, VARIANT_SUCCESS
from synnax.task.payload import Payload, Status, ontology_id
from synnax.telem import TimeSpan, TimeStamp
from synnax.util.normalize import check_for_none, normalize, override


class _CreateRequest(BaseModel):
    tasks: list[Payload]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[int]


class _CopyRequest(BaseModel):
    key: int
    name: str
    snapshot: bool


class _CopyResponse(BaseModel):
    task: Payload


class _RetrieveRequest(BaseModel):
    rack: int | None = None
    keys: list[int] | None = None
    names: list[str] | None = None
    types: list[str] | None = None
    include_status: bool = False
    internal: bool | None = None
    snapshot: bool | None = None


class _RetrieveResponse(BaseModel):
    tasks: list[Payload] | None = None


_CREATE_ENDPOINT = "/task/create"
_DELETE_ENDPOINT = "/task/delete"
_RETRIEVE_ENDPOINT = "/task/retrieve"
_COPY_ENDPOINT = "/task/copy"

_TASK_STATE_CHANNEL = "sy_status_set"
_TASK_CMD_CHANNEL = "sy_task_cmd"


class BaseTaskConfig(BaseModel):
    """
    Base configuration shared by all hardware task types.

    This base class provides common fields that all hardware integration tasks need:
    auto-start behavior.
    """

    auto_start: bool = False


class BaseReadTaskConfig(BaseTaskConfig):
    """
    Base configuration for hardware read/acquisition tasks.

    Extends BaseTaskConfig with sample rate and stream rate fields common to
    all data acquisition tasks (LabJack, NI, Modbus, OPC UA read tasks).

    Default rate limits are set to 50 kHz based on NI hardware constraints,
    which are the most restrictive across supported hardware platforms.
    Hardware-specific configs can override these limits for devices that
    support higher rates.
    """

    data_saving: bool = True
    "Whether to persist acquired data to disk (True) or only stream it (False)."
    sample_rate: conint(ge=0, le=50000)
    "The rate at which to sample data from the hardware device (Hz)."
    stream_rate: conint(ge=0, le=50000)
    "The rate at which acquired data will be streamed to the Synnax cluster (Hz)."

    @field_validator("stream_rate")
    def validate_stream_rate(cls, v, info):
        """Validate that stream_rate is less than or equal to sample_rate."""
        if "sample_rate" in info.data and v > info.data["sample_rate"]:
            raise ValueError(
                "Stream rate must be less than or equal to the sample rate"
            )
        return v


class BaseWriteTaskConfig(BaseTaskConfig):
    """
    Base configuration for hardware write/control tasks.

    Provides common fields (device, auto_start) for all hardware write tasks.
    Note that state_rate and data_saving are NOT included in this base class as they
    are hardware-specific - only write tasks with state feedback (NI, LabJack) use
    these fields. Tasks without state feedback (Modbus, OPC UA) do not need them.
    """

    device: str = Field(min_length=1)
    "The key of the Synnax device this task will communicate with."


class Task:
    key: int = 0
    name: str = ""
    type: str = ""
    config: str = ""
    snapshot: bool = False
    status: Status | None = None
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
        status: Status | None = None,
        _frame_client: FrameClient | None = None,
    ):
        if key == 0:
            key = (rack << 32) + 0
        self.key = key
        self.name = name
        self.type = type
        self.config = config
        self.snapshot = snapshot
        self.status = status
        self._frame_client = _frame_client

    def to_payload(self) -> Payload:
        return Payload(
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

    @property
    def ontology_id(self) -> dict:
        """Get the ontology ID for the task.

        Returns:
            An ontology ID dictionary with type "task" and the task key.
        """
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
                try:
                    status = Status.model_validate(frame[_TASK_STATE_CHANNEL][0])
                    if status.details.cmd is not None and status.details.cmd == key:
                        return status
                except ValidationError as e:
                    raise UnexpectedError(f"""
                    Received invalid task state from driver.
                    """) from e


class TaskProtocol(Protocol):
    key: int

    def to_payload(self) -> Payload: ...

    def set_internal(self, task: Task): ...

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

    def start(self, timeout: float | TimeSpan = 5):
        """Starts the task and blocks until the Synnax cluster has acknowledged the
        command or the specified timeout has elapsed.

        :raises TimeoutError: If the timeout is reached before the Synnax cluster
            acknowledges the command.
        :raises Exception: If the Synnax cluster fails to start the task correctly.
        """
        self._internal.execute_command_sync("start", timeout=timeout)

    def stop(self, timeout: float | TimeSpan = 5):
        """Stops the task and blocks until the Synnax cluster has acknowledged the
        command or the specified timeout has elapsed.

        :raises TimeoutError: If the timeout is reached before the Synnax cluster
            acknowledges the command.
        :raises Exception: If the Synnax cluster fails to stop the task correctly.
        """
        self._internal.execute_command_sync("stop", timeout=timeout)

    @contextmanager
    def run(self, timeout: float | TimeSpan = 5):
        """Context manager that starts the task before entering the block and stops the
        task after exiting the block. This is useful for ensuring that the task is
        properly stopped even if an exception occurs during execution.
        """
        self.start(timeout)
        try:
            yield
        finally:
            self.stop(timeout)


class JSONConfigMixin(TaskProtocol):
    _internal: Task
    config: BaseModel

    @property
    def name(self) -> str:
        return self._internal.name

    @property
    def key(self) -> int:
        """Implements TaskProtocol protocol"""
        return self._internal.key

    def to_payload(self) -> Payload:
        """Implements TaskProtocol protocol"""
        pld = self._internal.to_payload()
        pld.config = json.dumps(self.config.model_dump())
        return pld

    def set_internal(self, task: Task):
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
        key: int = 0,
        name: str = "",
        type: str = "",
        config: str = "",
        rack: int = 0,
    ): ...

    @overload
    def create(self, tasks: Task) -> Task: ...

    @overload
    def create(self, tasks: list[Task]) -> list[Task]: ...

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
            tasks = [Payload(key=key, name=name, type=type, config=config)]
        elif isinstance(tasks, Task):
            tasks = [tasks.to_payload()]
        else:
            is_single = False
            tasks = [t.to_payload() for t in tasks]
        for pld in tasks:
            self.maybe_assign_def_rack(pld, rack)
        req = _CreateRequest(tasks=tasks)
        tasks = self.__exec_create(req)
        sugared = self.sugar(tasks)
        return sugared[0] if is_single else sugared

    def __exec_create(self, req: _CreateRequest) -> list[Payload]:
        res = send_required(self._client, "/task/create", req, _CreateResponse)
        return res.tasks

    def maybe_assign_def_rack(self, pld: Payload, rack: int = 0) -> Rack:
        if self._default_rack is None:
            # Hardcoded as this value for now. Will be changed once we have multi-rack
            # systems
            self._default_rack = self._racks.retrieve_embedded_rack()
        if pld is not None and pld.key == 0:
            if rack == 0:
                rack = self._default_rack.key
            pld.key = (rack << 32) + 0
        return pld

    def configure(self, task: TaskProtocol, timeout: float = 5) -> TaskProtocol:
        # Call task-specific device property update (e.g., for Modbus, OPC UA, LabJack)
        if self._device_client is not None:
            task.update_device_properties(self._device_client)

        with self._frame_client.open_streamer([_TASK_STATE_CHANNEL]) as streamer:
            pld = self.maybe_assign_def_rack(task.to_payload())
            req = _CreateRequest(tasks=[pld])
            tasks = self.__exec_create(req)
            task.set_internal(self.sugar(tasks)[0])
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
                status = Status.model_validate(frame[_TASK_STATE_CHANNEL][0])
                if status.details.task != task.key:
                    continue
                if status.variant == VARIANT_SUCCESS:
                    break
                if status.variant == VARIANT_ERROR:
                    raise ConfigurationError(status.message)
        return task

    def delete(self, keys: int | list[int]):
        req = _DeleteRequest(keys=normalize(keys))
        send_required(self._client, "/task/delete", req, Empty)

    @overload
    def retrieve(
        self,
        *,
        key: int | None = None,
        name: str | None = None,
        type: str | None = None,
    ) -> Task: ...

    @overload
    def retrieve(
        self,
        names: list[str] | None = None,
        keys: list[int] | None = None,
        types: list[str] | None = None,
    ) -> list[Task]: ...

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
            "/task/retrieve",
            _RetrieveRequest(
                keys=override(key, keys),
                names=override(name, names),
                types=override(type, types),
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

    def list(self, rack: int | None = None) -> list[Task]:
        """Lists all tasks on a rack. If no rack is specified, lists all tasks on the
        default rack. Excludes internal system tasks (scanner tasks and rack state).

        :param rack: The rack key to list tasks from. If None, uses the default rack.
        :return: A list of all user-created tasks on the specified rack.
        """
        if rack is None and self._default_rack is None:
            self._default_rack = self._racks.retrieve_embedded_rack()
        if rack is None:
            rack = self._default_rack.key

        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(rack=rack, internal=False),
            _RetrieveResponse,
        )
        return self.sugar(res.tasks)

    def copy(
        self,
        key: int,
        name: str,
    ) -> Task:
        """Copies an existing task with a new name.

        :param key: The key of the task to copy.
        :param name: The name for the new task.
        :return: The newly created task.
        """
        req = _CopyRequest(key=key, name=name, snapshot=False)
        res = send_required(self._client, _COPY_ENDPOINT, req, _CopyResponse)
        return self.sugar([res.task])[0]

    def sugar(self, tasks: list[Payload]):
        return [Task(**t.model_dump(), _frame_client=self._frame_client) for t in tasks]
