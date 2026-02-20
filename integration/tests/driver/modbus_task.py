#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus-specific task test cases."""

from abc import abstractmethod

import synnax as sy
from examples.modbus import ModbusSim

from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase


class ModbusReadTaskCase(SimulatorCase, ReadTaskCase):
    """Base class for Modbus TCP read task tests."""

    sim_classes = [ModbusSim]
    SAMPLE_RATE = 50 * sy.Rate.HZ

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.modbus.ReadTask:
        """Create a Modbus read task."""
        channels = self.create_channels(self.client)

        return sy.modbus.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            data_saving=True,
            channels=channels,
        )


class ModbusWriteTaskCase(SimulatorCase, WriteTaskCase):
    """Base class for Modbus TCP write task tests."""

    sim_classes = [ModbusSim]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.OutputChan]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.modbus.WriteTask:
        """Create a Modbus write task."""
        channels = self.create_channels(self.client)
        return sy.modbus.WriteTask(
            name=task_name,
            device=device.key,
            channels=channels,
        )
