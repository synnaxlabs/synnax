#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Modbus-specific task test case.

Provides Modbus TCP task creation logic using Synnax task client directly.
"""

from abc import abstractmethod
from typing import Any

import synnax as sy
from synnax import modbus
from synnax.modbus.types import BaseChan

from driver.devices import Simulator
from tests.driver.simulator_task import SimulatorTaskCase


class ModbusTaskCase(SimulatorTaskCase):
    """
    Base class for Modbus TCP task tests.

    Provides Modbus-specific task creation using Synnax task channels directly.
    Subclasses should implement create_channels() to define task-specific channels.
    """

    def __init__(
        self,
        *,
        task_name: str,
        **kwargs: Any,
    ) -> None:
        """
        Initialize ModbusTaskCase.

        The device_name is automatically set from the Modbus simulator configuration.
        """
        super().__init__(
            task_name=task_name,
            simulator=Simulator.MODBUS,
            **kwargs,
        )

    @abstractmethod
    def create_channels(self) -> list[BaseChan]:
        """Create Modbus-specific task channels.

        Returns:
            List of Modbus channel objects (e.g., InputRegisterChan, HoldingRegisterInputChan)
        """
        pass

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> modbus.ReadTask:
        """
        Create a Modbus read task.

        Args:
            device: Synnax device to read from
            task_name: Name for the task
            sample_rate: Sampling rate for the task
            stream_rate: Streaming rate for the task

        Returns:
            Configured Modbus read task
        """
        channels = self.create_channels()

        return modbus.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=channels,
        )
