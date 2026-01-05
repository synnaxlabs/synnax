#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
OPC UA-specific task test case.

Provides OPC UA task creation logic using Synnax task client directly.
"""

from abc import abstractmethod
from typing import Any

import synnax as sy
from synnax import opcua

from driver.devices import Simulator
from tests.driver.simulator_task import SimulatorTaskCase


class OPCUATaskCase(SimulatorTaskCase):
    """
    Base class for OPC UA task tests.

    Provides OPC UA-specific task creation using Synnax task channels directly.
    Subclasses should implement create_channels() to define task-specific channels.
    """

    def __init__(
        self,
        *,
        task_name: str,
        array_mode: bool = False,
        array_size: int = 5,
        **kwargs: Any,
    ) -> None:
        """
        Initialize OPCUATaskCase.

        The device_name is automatically set from the OPC UA simulator configuration.
        """
        self.array_mode: bool = array_mode
        self.array_size: int = array_size

        super().__init__(
            task_name=task_name,
            simulator=Simulator.OPCUA,
            **kwargs,
        )

    @abstractmethod
    def create_channels(self) -> list[opcua.ReadChannel]:
        """Create OPC UA-specific task channels.

        Returns:
            List of OPC UA ReadChannel objects
        """
        pass

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> opcua.ReadTask:
        """
        Create an OPC UA read task.

        Args:
            device: Synnax device to read from
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured OPC UA read task
        """
        channels = self.create_channels()

        return opcua.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            array_mode=self.array_mode,
            array_size=self.array_size,
            channels=channels,
        )
