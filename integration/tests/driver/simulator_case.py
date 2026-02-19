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

Single simulator usage:

    class TaskToolbar(SimulatorCase, ConsoleCase):
        sim_class = OPCUASim

Multiple simulator usage:

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

    Subclasses set either sim_class (single) or sim_classes (multiple).
    """

    sim_class: type[DeviceSim]
    sim_classes: list[type[DeviceSim]] = []
    sim: DeviceSim | None = None
    sims: dict[str, DeviceSim | None]
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")

    def setup(self) -> None:
        """Start simulator(s), connect device(s), then delegate to next in MRO."""
        self.sims = getattr(self, "sims", {})
        if self.sim_classes:
            self._setup_multi_sims()
        else:
            self._setup_single_sim()
        super().setup()

    def _setup_single_sim(self) -> None:
        """Original single-simulator setup path."""
        self.device_name = self.sim_class.device_name
        if self.sim is None:
            self.sim = self.sim_class(rate=self.SAMPLE_RATE)
        self.sim.start()
        self.sims[self.sim_class.device_name] = self.sim
        self._connect_device()

    def _setup_multi_sims(self) -> None:
        """Start multiple simulators and register their devices."""
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

    def start_simulator(self, device_name: str | None = None) -> None:
        """Start (or restart) a simulator.

        Args:
            device_name: Target a specific sim in multi-sim mode. When None,
                         uses the original single-sim behavior.
        """
        if device_name is not None and device_name in self.sims:
            old_sim = self.sims[device_name]
            if old_sim is not None:
                old_sim.stop()
            sim_cls = next(c for c in self.sim_classes if c.device_name == device_name)
            new_sim = sim_cls(rate=self.SAMPLE_RATE)
            new_sim.start()
            self.sims[device_name] = new_sim
            if device_name == self.device_name:
                self.sim = new_sim
        else:
            if self.sim is not None:
                self.sim.stop()
            self.sim = self.sim_class(rate=self.SAMPLE_RATE)
            self.sim.start()
            self.sims[self.sim_class.device_name] = self.sim

    def cleanup_simulator(
        self, log: bool = False, device_name: str | None = None
    ) -> None:
        """Stop a simulator.

        Args:
            log: Whether to log cleanup.
            device_name: Target a specific sim in multi-sim mode. When None,
                         uses the original single-sim behavior.
        """
        if device_name is not None and device_name in self.sims:
            sim = self.sims.get(device_name)
            if sim is not None:
                sim.stop()
                self.sims[device_name] = None
        else:
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
        for sim in self.sims.values():
            if sim is not None:
                sim.stop()
        self.sims = {}
        self.sim = None

    def _connect_device(self) -> None:
        """Get or create the hardware device for the single sim_class."""
        self._connect_device_for(self.sim_class)

    def _connect_device_for(self, sim_cls: type[DeviceSim]) -> None:
        """Get or create the hardware device for a given simulator class."""
        rack = self.client.racks.retrieve(name=self.RACK_NAME)
        device_instance = sim_cls.create_device(rack.key)
        try:
            self.client.devices.retrieve(name=device_instance.name)
        except sy.NotFoundError:
            self.client.devices.create(device_instance)
