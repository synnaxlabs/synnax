#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Simulator-based task test case.

Extends TaskCase with simulator server lifecycle management for protocols
that require simulated hardware (Modbus, OPC UA).
"""

from multiprocessing.process import BaseProcess
from typing import Any

import synnax as sy
from examples.simulators.device_sim import DeviceSim

from tests.driver.task import TaskCase


class SimulatorTaskCase(TaskCase):
    """
    Base class for driver task tests that require a device simulator.

    Adds simulator lifecycle management (start/stop server) on top of TaskCase.
    Use this for protocols that need simulated hardware (Modbus, OPC UA).

    Subclasses must:
    - Set sim_class as a class attribute (a DeviceSim subclass)
    - All other requirements from TaskCase still apply

    The device_name is automatically set from sim_class.device_name.
    Device registration uses sim_class.create_device().
    """

    sim_class: type[DeviceSim]
    sim: DeviceSim | None = None

    def __init__(
        self,
        *,
        task_name: str,
        **params: Any,
    ) -> None:
        super().__init__(
            task_name=task_name,
            **params,
        )
        if not hasattr(self, "sim_class") or self.sim_class is None:
            raise TypeError(
                f"{self.__class__.__name__} must define 'sim_class' class attribute"
            )
        self.device_name = self.sim_class.device_name

    def setup(self) -> None:
        """Start simulator, connect device, and configure task."""
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

    def _connect_device(
        self,
        max_retries: int = 10,
        retry_delay: float = 1.0,
    ) -> None:
        """Get or create the hardware device for this simulator."""
        for attempt in range(max_retries):
            try:
                rack = self.client.racks.retrieve(name=self.RACK_NAME)
                break
            except Exception as e:
                if attempt < max_retries - 1:
                    sy.sleep(retry_delay)
                else:
                    raise AssertionError(
                        f"Failed to retrieve rack '{self.RACK_NAME}' "
                        f"after {max_retries} attempts: {e}"
                    )
        else:
            raise AssertionError(
                f"Rack '{self.RACK_NAME}' not found after {max_retries} attempts"
            )

        device_instance = self.sim_class.create_device(rack.key)

        try:
            device = self.client.devices.retrieve(name=device_instance.name)
        except sy.NotFoundError:
            device = self.client.devices.create(device_instance)
        except Exception as e:
            raise AssertionError(f"Unexpected error creating device: {e}")

        return
