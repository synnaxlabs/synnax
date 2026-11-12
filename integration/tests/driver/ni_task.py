#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI-specific task test cases."""

import ctypes
import pathlib
import platform
from abc import abstractmethod
from collections.abc import Callable, Sequence

import synnax as sy
from framework import resources
from tests.driver.task import (
    ReadTaskCase,
    ReadTaskT,
    TaskCase,
    TaskT,
    WriteTaskCase,
)

_NI_MAX_CONFIG = pathlib.Path(__file__).parent.parent / "fixtures" / "ni_max_config.nce"


def _ensure_ni_max_config(log: Callable[[str], None]) -> None:
    """Import the reference NI MAX config unconditionally.

    Always re-imports to guarantee NI MAX state is correct, even if a previous
    test (e.g. a disconnect test) deleted devices without restoring them.

    Only meaningful on Windows with NI System Configuration installed.
    """
    if not _NI_MAX_CONFIG.exists():
        log("ni_max_config.nce not found in fixtures, skipping config sync")
        return

    import nisyscfg  # Windows-only; caller guards with platform check

    log("Importing NI MAX config ...")
    with nisyscfg.Session() as session:
        detailed_result = ctypes.POINTER(ctypes.c_char)()
        status = session._library.ImportConfiguration(
            session._session,
            str(_NI_MAX_CONFIG).encode(),
            b"",
            nisyscfg.enums.ImportMode.MERGE_ITEMS,
            ctypes.byref(detailed_result),
        )
    if status != 0:
        raise RuntimeError(f"NISysCfgImportConfiguration failed with status {status}")
    log("NI MAX config imported successfully")


# The scan task polls every 5 s.
_PRESENT_TIMEOUT = 30 * sy.TimeSpan.SECOND


def retrieve_present_device(client: sy.Synnax, location: str) -> sy.Device:
    """Return the device at an NI MAX location that the Driver reports as present.

    The Driver keeps a device that it no longer sees, so after an NI MAX change a
    location can hold a disconnected device next to the present one.

    :raises TimeoutError: If the location does not hold exactly one present device
        within 30 seconds.
    """
    found = ""

    def present() -> sy.Device | None:
        nonlocal found
        devices = client.devices.retrieve(location=location, include_status=True)
        matches = [
            d for d in devices if d.status is not None and d.status.variant == "success"
        ]
        found = f"{len(matches)} of {len(devices)}"
        return matches[0] if len(matches) == 1 else None

    device = sy.poll(present, timeout=_PRESENT_TIMEOUT, interval=0.5)
    if device is None:
        raise TimeoutError(
            f"Expected one present device at {location}, but found {found}"
        )
    return device


class _NITaskMixin(TaskCase[TaskT]):
    """Shared setup for all NI task tests.

    Skips on non-Windows platforms. The NI driver scanner discovers devices and
    registers them with location = NI MAX alias (e.g. "E101Mod1").

    Subclasses must set:
        device_locations: list[str]  — NI MAX identifiers (e.g. ["E101Mod1"])
    """

    device_locations: list[str] = []
    # False for real devices, whose machine must keep its NI MAX configuration.
    simulated: bool = True
    devices: dict[str, sy.Device]

    @classmethod
    def resources(cls) -> frozenset[str]:
        return super().resources() | {resources.NI}

    @property
    def _device_key(self) -> str:
        """Return 'cross-device' for multi-device tasks, else the single device key."""
        if len(self.device_locations) > 1:
            return "cross-device"
        key: str = self.devices[self.device_locations[0]].key
        return key

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.skip("Windows DAQmx drivers required")
        if self.simulated:
            _ensure_ni_max_config(self.log)
        # The NI scanner registers devices with location = NI MAX alias.
        # Resolve all locations upfront so concrete tests can use
        # devices[location] in create_channels without additional retrieves.
        self.devices = {
            loc: retrieve_present_device(self.client, loc)
            for loc in self.device_locations
        }
        super().setup()

    def retrieve_device(self) -> sy.Device:
        # Every NI device of one model shares a name, so use the first location.
        return self.devices[self.device_locations[0]]


class _NIReadTaskBase(_NITaskMixin[ReadTaskT], ReadTaskCase[ReadTaskT]):
    """Shared create() for all NI read task types.

    Subclasses set ``_task_class`` to the concrete sy.ni task type and
    implement ``create_channels`` with the appropriate channel-type annotation.
    """

    _task_class: Callable[..., ReadTaskT]

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[object]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> ReadTaskT:
        channels = self.create_channels(self.client, self.devices)
        return self._task_class(
            name=task_name,
            device=self._device_key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            channels=channels,
        )


class _NIWriteTaskBase(_NITaskMixin[TaskT], WriteTaskCase[TaskT]):
    """Shared create() for all NI write task types.

    Subclasses set ``_task_class`` to the concrete sy.ni task type and
    implement ``create_channels`` with the appropriate channel-type annotation.
    """

    _task_class: Callable[..., TaskT]

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[object]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> TaskT:
        channels = self.create_channels(self.client, self.devices)
        return self._task_class(
            name=task_name,
            device=self._device_key,
            state_rate=sample_rate,
            channels=channels,
        )

    @abstractmethod
    def _state_keys(self, task: TaskT) -> list[int]: ...


class NIAnalogReadTaskCase(_NIReadTaskBase[sy.ni.AnalogReadTask]):
    """Base class for NI analog read task tests."""

    _task_class: Callable[..., sy.ni.AnalogReadTask] = sy.ni.AnalogReadTask

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[sy.ni.AIChannel]: ...

    def _channel_keys(self, task: sy.ni.AnalogReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]


class NIDigitalReadTaskCase(_NIReadTaskBase[sy.ni.DigitalReadTask]):
    """Base class for NI digital read task tests."""

    _task_class: Callable[..., sy.ni.DigitalReadTask] = sy.ni.DigitalReadTask

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[sy.ni.DIChannel]: ...

    def _channel_keys(self, task: sy.ni.DigitalReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]


class NICounterReadTaskCase(_NIReadTaskBase[sy.ni.CounterReadTask]):
    """Base class for NI counter read task tests."""

    _task_class: Callable[..., sy.ni.CounterReadTask] = sy.ni.CounterReadTask

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[sy.ni.CIChannel]: ...

    def _channel_keys(self, task: sy.ni.CounterReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]


class NIAnalogWriteTaskCase(_NIWriteTaskBase[sy.ni.AnalogWriteTask]):
    """Base class for NI analog write task tests."""

    _task_class: Callable[..., sy.ni.AnalogWriteTask] = sy.ni.AnalogWriteTask

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[sy.ni.AOChannel]: ...

    def _channel_keys(self, task: sy.ni.AnalogWriteTask) -> list[int]:
        return [
            ch.cmd_channel
            for ch in task.config.channels
            if not isinstance(ch, sy.ni.AOFuncGenChannel)
        ]

    def _state_keys(self, task: sy.ni.AnalogWriteTask) -> list[int]:
        return [
            ch.state_channel
            for ch in task.config.channels
            if not isinstance(ch, sy.ni.AOFuncGenChannel)
        ]


class NIDigitalWriteTaskCase(_NIWriteTaskBase[sy.ni.DigitalWriteTask]):
    """Base class for NI digital write task tests."""

    _task_class: Callable[..., sy.ni.DigitalWriteTask] = sy.ni.DigitalWriteTask

    @staticmethod
    @abstractmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> Sequence[sy.ni.DOChannel]: ...

    def _channel_keys(self, task: sy.ni.DigitalWriteTask) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    def _state_keys(self, task: sy.ni.DigitalWriteTask) -> list[int]:
        return [ch.state_channel for ch in task.config.channels]
