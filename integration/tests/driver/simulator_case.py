#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Simulator lifecycle mixin.

Provides DeviceSim server management (start/stop) and device registration
in Synnax. Designed for multiple inheritance with TestCase subclasses.

Usage standalone (e.g., with ConsoleCase):

    class TaskToolbar(SimulatorCase, ConsoleCase):
        sim_class = OPCUASim`

Usage with TaskCase:

    class ModbusTaskCase(SimulatorCase, TaskCase):
        sim_class = ModbusSim
"""

import os
from multiprocessing.process import BaseProcess

import synnax as sy
from examples.simulators.device_sim import DeviceSim

from framework.test_case import TestCase


class SimulatorCase(TestCase):
    """DeviceSim lifecycle management.

    Subclasses must set sim_class as a class attribute.
    """

    sim_class: type[DeviceSim]
    sim: DeviceSim | None = None
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")

    def setup(self) -> None:
        """Start simulator, connect device, then delegate to next in MRO."""
        self.device_name = self.sim_class.device_name
        if self.sim is None:
            self.sim = self.sim_class(rate=self.SAMPLE_RATE)
        self.sim.start()
        self._connect_device()
        super().setup()

    def start_simulator(self) -> None:
        """Start (or restart) the simulator."""
        if self.sim is not None:
            self.sim.stop()
        self.sim = self.sim_class(rate=self.SAMPLE_RATE)
        self.sim.start()

    def cleanup_simulator(self, log: bool = False) -> None:
        """Stop the simulator."""
        if self.sim is not None:
            self.sim.stop()
            self.sim = None

    @property
    def simulator_process(self) -> BaseProcess | None:
        """Access the underlying process (for DisconnectTask compatibility)."""
        if self.sim is not None:
            return self.sim.process
        return None

    def teardown(self) -> None:
        """Cleanup after test."""
        super().teardown()
        self.cleanup_simulator(log=True)

    def _connect_device(self) -> None:
        """Get or create the hardware device for this simulator."""
        rack = self.client.racks.retrieve(name=self.RACK_NAME)
        device_instance = self.sim_class.create_device(rack.key)
        try:
            self.client.devices.retrieve(name=device_instance.name)
        except sy.NotFoundError:
            self.client.devices.create(device_instance)
