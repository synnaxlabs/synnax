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

    class CardRead(SimulatorCase):
        device_classes = [ARINC429Card]
"""

import os
import subprocess
from multiprocessing.process import BaseProcess
from typing import Protocol

import psutil
from examples.simulators.device_sim import DeviceSim

import synnax as sy
from framework.hardware_case import HardwareCase
from framework.models import SynnaxConnection


def _listener_pids(port: int) -> list[int]:
    try:
        return [
            conn.pid
            for conn in psutil.net_connections(kind="tcp")
            if conn.status == "LISTEN"
            and conn.laddr
            and conn.laddr.port == port
            and conn.pid
        ]
    except psutil.AccessDenied:
        # macOS hides other processes' sockets from non-root users.
        try:
            out = subprocess.run(
                ["lsof", "-nP", f"-iTCP:{port}", "-sTCP:LISTEN", "-t"],
                capture_output=True,
                text=True,
            ).stdout
        except OSError:
            return []
        return [int(pid) for pid in out.split()]


class DeviceSpec(Protocol):
    """A device a case registers in Synnax. A DeviceSim is one; a device that needs no
    simulator process, such as a card the Driver simulates, is another."""

    device_name: str
    channel_names: tuple[str, ...]

    @staticmethod
    def create_device(rack_key: int) -> sy.Device: ...


class SimulatorCase(HardwareCase):
    """DeviceSim lifecycle management.

    Subclasses set sim_classes to a list of DeviceSim subclasses, and device_classes to
    the devices to register without a simulator. The first sim, or the first device
    when there is no sim, is the primary one (self.sim / self.device_name).
    """

    sim_classes: list[type[DeviceSim]] = []
    device_classes: list[type[DeviceSpec]] = []
    sim: DeviceSim | None = None
    sims: dict[str, DeviceSim | None]
    SAMPLE_RATE: sy.Rate = 50 * sy.Rate.HZ
    RACK_NAME: str = os.environ.get("SYNNAX_DRIVER_RACK", "Node 1 Embedded Driver")

    def __init__(
        self,
        synnax_connection: SynnaxConnection = SynnaxConnection(),
        *,
        name: str,
        **params: object,
    ) -> None:
        super().__init__(synnax_connection, name=name, **params)
        self.sims = {}

    def setup(self) -> None:
        """Start simulator(s), connect device(s), then delegate to next in MRO."""
        for sim_cls in self.sim_classes:
            name = sim_cls.device_name
            existing = self.sims.get(name)
            sim = existing if existing is not None else sim_cls(rate=self.SAMPLE_RATE)
            self._free_port(sim)
            sim.start()
            self.sims[name] = sim
            self._connect_device_for(sim_cls)
            self._reclaim_channels(sim_cls)
        for device_cls in self.device_classes:
            self._connect_device_for(device_cls)
            self._reclaim_channels(device_cls)
        if self.sim_classes:
            self.device_name = self.sim_classes[0].device_name
            self.sim = self.sims[self.device_name]
        else:
            self.device_name = self.device_classes[0].device_name
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
        """Stop simulator(s) first, then run the remaining teardown."""
        try:
            for sim in self.sims.values():
                if sim is not None:
                    with self._try_to("stop simulator"):
                        sim.stop()
            self.sims = {}
            self.sim = None
        finally:
            super().teardown()

    def _free_port(self, sim: DeviceSim) -> None:
        """Kill a stale process, usually left by a killed run, on the sim's port."""
        for pid in _listener_pids(sim.port):
            try:
                proc = psutil.Process(pid)
                proc.kill()
                proc.wait(timeout=5)
                self.log(f"Killed stale PID {pid} on port {sim.port}")
            except (psutil.NoSuchProcess, psutil.AccessDenied) as e:
                self.log(f"Could not kill PID {pid} on port {sim.port}: {e}")

    def _connect_device_for(self, device_cls: type[DeviceSpec]) -> None:
        """Get or create the hardware device of a device class."""
        rack = self.client.racks.retrieve(name=self.RACK_NAME)
        device_instance = device_cls.create_device(rack.key)
        try:
            existing = self.client.devices.retrieve(name=device_instance.name)
            # A sim device only exists for its sim, so one found here is a leftover from
            # a killed teardown; adopt it so this run's teardown reclaims it.
            self.track_test_devices([existing])
        except sy.NotFoundError:
            self.create_test_devices([device_instance])

    def _reclaim_channels(self, device_cls: type[DeviceSpec]) -> None:
        """Delete channels an earlier test left behind under the device's names."""
        if not device_cls.channel_names:
            return
        try:
            self.client.channels.delete(list(device_cls.channel_names))
        except sy.NotFoundError:
            pass
