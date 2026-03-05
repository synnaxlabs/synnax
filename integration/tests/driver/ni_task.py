#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI-specific task test cases."""

import platform
from abc import abstractmethod

import synnax as sy

from tests.driver.task import ReadTaskCase, WriteTaskCase


class _NITaskMixin:
    """Shared setup for all NI task tests.

    Auto-passes on non-Windows platforms. The NI driver scanner
    automatically discovers devices and registers them with
    location = NI MAX alias (e.g. "E101Mod1").

    Subclasses must set:
        device_name: str  — the NI MAX identifier (e.g. "E101Mod1")
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="Windows DAQmx drivers required")
        # The NI scanner registers devices with location = NI MAX alias,
        # but TaskCase.setup() retrieves by name. Resolve here.
        dev = self.client.devices.retrieve(location=self.device_name)
        self.device_name = dev.name
        super().setup()


class NIAnalogReadTaskCase(_NITaskMixin, ReadTaskCase):
    """Base class for NI analog read task tests."""

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.AIChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ni.AnalogReadTask:
        """Create an NI analog read task."""
        channels = self.create_channels(self.client)

        return sy.ni.AnalogReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=channels,
        )


class NIDigitalReadTaskCase(_NITaskMixin, ReadTaskCase):
    """Base class for NI digital read task tests."""

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.DIChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ni.DigitalReadTask:
        """Create an NI digital read task."""
        channels = self.create_channels(self.client)

        return sy.ni.DigitalReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=channels,
        )


class NICounterReadTaskCase(_NITaskMixin, ReadTaskCase):
    """Base class for NI counter read task tests."""

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.CIChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ni.CounterReadTask:
        """Create an NI counter read task."""
        channels = self.create_channels(self.client)

        return sy.ni.CounterReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=channels,
        )


class NIAnalogWriteTaskCase(_NITaskMixin, WriteTaskCase):
    """Base class for NI analog write task tests."""

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.AOChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ni.AnalogWriteTask:
        """Create an NI analog write task."""
        channels = self.create_channels(self.client)
        return sy.ni.AnalogWriteTask(
            name=task_name,
            device=device.key,
            state_rate=sample_rate,
            channels=channels,
        )


class NIDigitalWriteTaskCase(_NITaskMixin, WriteTaskCase):
    """Base class for NI digital write task tests."""

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.ni.DOChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.ni.DigitalWriteTask:
        """Create an NI digital write task."""
        channels = self.create_channels(self.client)
        return sy.ni.DigitalWriteTask(
            name=task_name,
            device=device.key,
            state_rate=sample_rate,
            channels=channels,
        )
