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

Usage:

    class MyTask(SimulatorCase, ReadTaskCase):
        sim_classes = [OPCUASim]

    class GrandFinale(SimulatorCase):
        sim_classes = [OPCUASim, ModbusSim]
"""

import os
from multiprocessing.process import BaseProcess

import synnax as sy
from examples.simulators.device_sim import DeviceSim

from framework.test_case import TestCase


class SimulatorCase(TestCase):
    """DeviceSim lifecycle management.

    Subclasses set sim_classes to a list of DeviceSim subclasses.
    The first entry is used as the primary sim (self.sim / self.device_name).
    """

    sim_classes: list[type[DeviceSim]] = []
    sim: DeviceSim | None = None
    sims: dict[str, DeviceSim | None]
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")

    def setup(self) -> None:
        """Start simulator(s), connect device(s), then delegate to next in MRO."""
        self.sims = getattr(self, "sims", {})
        for sim_cls in self.sim_classes:
            name = sim_cls.device_name
            existing = self.sims.get(name)
            sim = existing if existing is not None else sim_cls(rate=self.SAMPLE_RATE)
            sim.start()
            self.sims[name] = sim
            self._connect_device_for(sim_cls)
        first_cls = self.sim_classes[0]
        self.sim = self.sims[first_cls.device_name]
        self.device_name = first_cls.device_name
        super().setup()

    def start_simulator(self, device_name: str | None = None) -> None:
        """Start (or restart) a simulator.

        Args:
            device_name: Target a specific sim. When None, restarts the primary sim.
        """
        target = device_name or self.device_name
        old_sim = self.sims.get(target)
        if old_sim is not None:
            old_sim.stop()
        sim_cls = next(c for c in self.sim_classes if c.device_name == target)
        new_sim = sim_cls(rate=self.SAMPLE_RATE)
        new_sim.start()
        self.sims[target] = new_sim
        if target == self.device_name:
            self.sim = new_sim

    def cleanup_simulator(
        self, log: bool = False, device_name: str | None = None
    ) -> None:
        """Stop a simulator.

        Args:
            log: Whether to log cleanup.
            device_name: Target a specific sim. When None, stops the primary sim.
        """
        target = device_name or self.device_name
        sim = self.sims.get(target)
        if sim is not None:
            sim.stop()
            self.sims[target] = None
        if target == self.device_name:
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
        for sim in self.sims.values():
            if sim is not None:
                sim.stop()
        self.sims = {}
        self.sim = None

    def _connect_device_for(self, sim_cls: type[DeviceSim]) -> None:
        """Get or create the hardware device for a given simulator class."""
        rack = self.client.racks.retrieve(name=self.RACK_NAME)
        device_instance = sim_cls.create_device(rack.key)
        try:
            self.client.devices.retrieve(name=device_instance.name)
        except sy.NotFoundError:
            self.client.devices.create(device_instance)
