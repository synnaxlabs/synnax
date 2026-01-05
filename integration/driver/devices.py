#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Device and simulator configurations for driver integration tests.

This module provides:
- KnownDevices: Registry of all test device configurations
- Simulator: Class with available simulator server configurations
- connect_device: Utility function to get or create hardware devices
"""

import asyncio
import multiprocessing
import os
import signal
import sys
from dataclasses import dataclass
from multiprocessing.process import BaseProcess
from typing import Callable

import synnax as sy
from synnax import modbus, opcua
from synnax.device import Device as SynnaxDevice


def _run_modbus_server() -> None:
    """Run Modbus server in a subprocess."""
    # Import here to avoid issues with pickling
    from examples.modbus import run_server

    # Suppress stdout/stderr
    sys.stdout = open(os.devnull, "w")
    sys.stderr = open(os.devnull, "w")

    # Reset signal handlers to defaults
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    if sys.platform != "win32":
        signal.signal(signal.SIGTERM, signal.SIG_DFL)

    asyncio.run(run_server())


def _run_opcua_server() -> None:
    """Run OPC UA server in a subprocess."""
    # Import here to avoid issues with pickling
    from examples.opcua import run_server

    # Suppress stdout/stderr
    sys.stdout = open(os.devnull, "w")
    sys.stderr = open(os.devnull, "w")

    # Reset signal handlers to defaults
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    if sys.platform != "win32":
        signal.signal(signal.SIGTERM, signal.SIG_DFL)

    asyncio.run(run_server())


class KnownDevices:
    """
    Registry of known test device configurations.

    Each device is a factory function that takes a rack key and returns a
    configured Synnax device instance (modbus.Device, opcua.Device, etc.).
    """

    @staticmethod
    def modbus_sim(rack_key: int) -> modbus.Device:
        """Modbus TCP simulator device configuration."""
        return modbus.Device(
            host="127.0.0.1",
            port=5020,
            name="Modbus TCP Test Server",
            location="127.0.0.1:5020",
            rack=rack_key,
            swap_bytes=False,
            swap_words=False,
        )

    @staticmethod
    def opcua_sim(rack_key: int) -> opcua.Device:
        """OPC UA simulator device configuration."""
        return opcua.Device(
            endpoint="opc.tcp://127.0.0.1:4841/freeopcua/server/",
            name="OPC UA Test Server",
            location="opc.tcp://127.0.0.1:4841/freeopcua/server/",
            rack=rack_key,
        )


@dataclass(frozen=True)
class SimulatorConfig:
    """
    Configuration for a simulator server.
    Combines server startup callback with a reference to a device from KnownDevices.
    """

    server_setup: Callable[[], BaseProcess]
    startup_delay_seconds: float
    device_factory: Callable[[int], SynnaxDevice]
    device_name: str


def start_modbus_server() -> BaseProcess:
    """Start the Modbus TCP simulator server in a separate process."""
    process = multiprocessing.Process(target=_run_modbus_server, daemon=True)
    process.start()
    return process


def start_opcua_server() -> BaseProcess:
    """Start the OPC UA simulator server in a separate process."""
    process = multiprocessing.Process(target=_run_opcua_server, daemon=True)
    process.start()
    return process


class Simulator:
    """Available simulator servers for driver testing."""

    MODBUS = SimulatorConfig(
        server_setup=start_modbus_server,
        startup_delay_seconds=5.0,
        device_factory=KnownDevices.modbus_sim,
        device_name="Modbus TCP Test Server",
    )

    OPCUA = SimulatorConfig(
        server_setup=start_opcua_server,
        startup_delay_seconds=3.0,
        device_factory=KnownDevices.opcua_sim,
        device_name="OPC UA Test Server",
    )


def connect_device(
    client: sy.Synnax,
    *,
    rack_name: str,
    device_factory: Callable[[int], SynnaxDevice],
    max_retries: int = 10,
    retry_delay: float = 1.0,
) -> sy.Device:
    """
    Get or create a hardware device using a factory from KnownDevices.

    This function can be called for any device, whether or not a simulator is
    running. This enables tests to connect to multiple devices or to hardware
    devices without simulators.

    Args:
        client: Synnax client instance
        rack_name: Name of the rack to connect the device to
        device_factory: A factory function from KnownDevices (e.g., KnownDevices.modbus_sim)
        max_retries: Maximum number of retries to wait for rack to be available
        retry_delay: Delay in seconds between retries

    Returns:
        The created or retrieved Synnax device

    Example:
        device = connect_device(
            client,
            rack_name="Node 1 Embedded Driver",
            device_factory=KnownDevices.modbus_sim
        )
    """
    # Wait for the embedded driver to register its rack
    for attempt in range(max_retries):
        try:
            rack = client.racks.retrieve(name=rack_name)
            break
        except Exception as e:
            if attempt < max_retries - 1:
                sy.sleep(retry_delay)
            else:
                raise AssertionError(
                    f"Failed to retrieve rack '{rack_name}' after {max_retries} attempts: {e}"
                )
    else:
        raise AssertionError(
            f"Rack '{rack_name}' not found after {max_retries} attempts"
        )

    # Create device instance to get its name
    device_instance = device_factory(rack.key)
    device_name = device_instance.name

    try:
        device = client.devices.retrieve(name=device_name)
    except sy.NotFoundError:
        device = client.devices.create(device_instance)
    except Exception as e:
        raise AssertionError(f"Unexpected error creating device: {e}")

    return device
