#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA-specific task test cases."""

from abc import abstractmethod

import synnax as sy
from examples.opcua import OPCUASim
from synnax import opcua

from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase


class OPCUAReadTaskCase(SimulatorCase, ReadTaskCase):
    """Base class for OPC UA read task tests."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 100 * sy.Rate.HZ
    array_mode: bool = False
    array_size: int = 100

    def setup(self) -> None:
        self.sims = {
            OPCUASim.device_name: OPCUASim(
                rate=self.SAMPLE_RATE, array_size=self.array_size
            )
        }
        super().setup()

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[opcua.ReadChannel]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> opcua.ReadTask:
        """Create an OPC UA read task."""
        channels = self.create_channels(self.client)

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


class OPCUAWriteTaskCase(SimulatorCase, WriteTaskCase):
    """Base class for OPC UA write task tests."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 100 * sy.Rate.HZ

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[opcua.WriteChannel]: ...

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> opcua.WriteTask:
        """Create an OPC UA write task."""
        channels = self.create_channels(self.client)
        return opcua.WriteTask(
            name=task_name,
            device=device.key,
            channels=channels,
        )
