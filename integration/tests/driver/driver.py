#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Base class for driver integration tests.

Provides common functionality for:
- Launching and managing simulator servers
- Creating and retrieving hardware devices
- Server cleanup and teardown
"""

import atexit
import subprocess
import sys
from abc import abstractmethod
from typing import Callable

import synnax as sy
from synnax.hardware.device import Device as SynnaxDevice

from framework.test_case import TestCase
from tests.driver.devices import Simulator


class Driver(TestCase):
    """
    Abstract base class for driver integration tests.

    Usage patterns:

    1. Simulator tests (Modbus/OPC UA):
       - Set simulator: Simulator.MODBUS or Simulator.OPCUA
       - In setup(), simulator starts automatically
       - Device created from simulator's device_factory

       Example:
           from tests.driver.driver import Driver
           from tests.driver.devices import Simulator

           class ModbusBasic(Driver):
               simulator = Simulator.MODBUS

    2. Hardware tests without simulator:
       - Leave simulator as None
       - Call self.connect_device(KnownDevices.my_device) in run()

       Example:
           from tests.driver.driver import Driver
           from tests.driver.devices import KnownDevices

           class NITest(Driver):
               def run(self):
                   device = self.connect_device(KnownDevices.ni_daq_6001)

    3. Existing hardware (already in cluster):
       - Leave simulator as None
       - Manually retrieve device using self.client.hardware.devices.retrieve()

    Subclasses must implement:
    - run(): Test execution logic
    """
    
    # Sim server process variables
    simulator: Simulator | None = None
    server_process: subprocess.Popen[bytes] | None = None

    def setup(self) -> None:
        """Start simulator and connect to device (if simulator configured)."""
        if self.simulator is not None:
            self._start_simulator()
            device = self.connect_device(self.simulator.value.device_factory)
            self.log(f"Device: {device.name} (key={device.key})")

    def _start_simulator(self) -> None:
        """Start the simulator server."""
        config = self.simulator.value
        server_script = self.repo_root / config.server_script

        if not server_script.exists():
            raise FileNotFoundError(f"Server script not found: {server_script}")

        # Launch the simulator server
        self.server_process = subprocess.Popen(
            [sys.executable, str(server_script)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.log(f"Server started with PID: {self.server_process.pid}")

        # Register cleanup handler for process exit scenarios
        atexit.register(self._cleanup_server)

        # Wait for server to start
        sy.sleep(config.startup_delay_seconds)

        if self.server_process.poll() is not None:
            # Server failed - capture output for debugging
            stdout, stderr = self.server_process.communicate()
            error_msg = stderr.decode() if stderr else "No error output"
            raise RuntimeError(
                f"Server failed to start (exit code: {self.server_process.returncode})\n{error_msg}"
            )

    @abstractmethod
    def run(self) -> None:
        """Execute the test-specific logic."""
        pass

    def connect_device(
        self, device_factory: Callable[[str], SynnaxDevice]
    ) -> sy.Device:
        """
        Get or create a hardware device using a factory from KnownDevices.

        This is a public method that can be called for any device, whether or not
        a simulator is running. This enables tests to connect to multiple devices
        or to hardware devices without simulators.

        Args:
            device_factory: A factory function from KnownDevices (e.g., KnownDevices.modbus_sim)

        Returns:
            The created or retrieved Synnax device

        Example:
            device = self.connect_device(KnownDevices.modbus_sim)
        """
        client = self.client
        rack = client.hardware.racks.retrieve(name="Node 1 Embedded Driver")

        # Create device instance to get its name
        device_instance = device_factory(rack.key)
        device_name = device_instance.name

        try:
            device = client.hardware.devices.retrieve(name=device_name)
            self.log(f"Found existing device: {device.name}")
        except:
            self.log(f"Creating new device: {device_name}")
            device = client.hardware.devices.create(device_instance)
            self.log(f"Created device: {device.name} (key={device.key})")

        return device

    def _cleanup_server(self, log: bool = False) -> None:
        """Terminate simulator server process (internal use only)."""

        if self.server_process.poll() is not None:
            self.log("Server already terminated")
            self.server_process = None
            return

        self.log("Terminating server...")
        try:
            self.server_process.terminate()
            try:
                self.server_process.wait(timeout=5 if log else 3)
                self.log("Server terminated successfully")

            except subprocess.TimeoutExpired:
                self.log("Server did not terminate gracefully, killing...")
                self.server_process.kill()
                self.server_process.wait(timeout=2 if log else 1)
                self.log("Server killed")

        except Exception as e:
            raise RuntimeError(f"Error during server cleanup: {e}")

        finally:
            self.server_process = None

    def teardown(self) -> None:
        """Terminate the simulator server if one was started."""
        if self.server_process is not None:
            self._cleanup_server()
