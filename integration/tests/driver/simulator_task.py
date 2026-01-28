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

import atexit
from multiprocessing.process import BaseProcess
from typing import Any

import synnax as sy

from driver.devices import SimulatorConfig, connect_device
from tests.driver.task import TaskCase


class SimulatorTaskCase(TaskCase):
    """
    Base class for driver task tests that require a simulator server.

    Adds simulator lifecycle management (start/stop server) on top of TaskCase.
    Use this for protocols that need simulated hardware (Modbus, OPC UA).

    Subclasses must:
    - Set simulator as a class attribute (SimulatorConfig instance)
    - All other requirements from TaskCase still apply

    The DEVICE_NAME will be automatically set from simulator.device_name.
    """

    # Set in start_simulator()
    simulator_process: BaseProcess | None = None

    def __init__(
        self,
        *,
        task_name: str,
        simulator: SimulatorConfig,
        **params: Any,
    ) -> None:
        """
        Initialize SimulatorTaskCase.

        The device_name is automatically set from the simulator configuration.
        """
        super().__init__(
            task_name=task_name,
            **params,
        )
        self.simulator: SimulatorConfig = simulator
        self.device_name = simulator.device_name

    def setup(self) -> None:
        """Start simulator, connect device, and configure task."""

        self.start_simulator()

        connect_device(
            client=self.client,
            rack_name=self.RACK_NAME,
            device_factory=self.simulator.device_factory,
        )

        super().setup()

    def start_simulator(self) -> None:
        """Start the simulator server using the configured callback."""
        # Call the server_setup callback to start the server
        self.simulator_process = self.simulator.server_setup()
        atexit.register(self.cleanup_simulator)

        self.log(f"Server started with PID: {self.simulator_process.pid}")
        sy.sleep(self.simulator.startup_delay_seconds)

        # Check if server crashed during startup
        if not self.simulator_process.is_alive():
            raise RuntimeError(
                f"Server failed to start (exit code: {self.simulator_process.exitcode})"
            )

    def cleanup_simulator(self, log: bool = False) -> None:
        """Terminate simulator server process."""

        if self.simulator_process is None:
            return

        if not self.simulator_process.is_alive():
            self.simulator_process = None
            return

        try:
            self.simulator_process.terminate()
            self.simulator_process.join(timeout=5 if log else 3)

            if self.simulator_process.is_alive():
                self.simulator_process.kill()
                self.simulator_process.join(timeout=2 if log else 1)
                if log:
                    self.log("Server killed")

        except Exception as e:
            if log:
                self.log(f"Error terminating simulator: {e}")
        finally:
            self.simulator_process = None
            # Give the OS time to release the port
            sy.sleep(1)

    def teardown(self) -> None:
        """Cleanup after test."""
        super().teardown()
        self.cleanup_simulator(log=True)
