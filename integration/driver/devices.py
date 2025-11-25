#  Copyright 2025 Synnax Labs, Inc.
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
"""

import asyncio
import os
import signal
import sys
from collections.abc import Coroutine
from dataclasses import dataclass
from multiprocessing import get_context
from multiprocessing.context import ForkProcess
from typing import Any, Callable

# isort: off
from examples.modbus import run_server as run_modbus_server  # type: ignore[import-untyped]
from examples.opcua import run_server as run_opcua_server  # type: ignore[import-untyped]

# isort: on
from synnax.hardware import modbus, opcua
from synnax.hardware.device import Device as SynnaxDevice

# Use fork method for multiprocessing to support lambdas
mp_ctx = get_context("fork")


def _run_server(server_func: Callable[[], Coroutine[Any, Any, None]]) -> None:
    """Run a server in a subprocess, with default signal handling."""

    # Suppress stdout
    sys.stdout = open(os.devnull, "w")

    signal.signal(signal.SIGINT, signal.SIG_DFL)
    signal.signal(signal.SIGTERM, signal.SIG_DFL)
    asyncio.run(server_func())


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
            endpoint="opc.tcp://localhost:4841/freeopcua/server/",
            name="OPC UA Test Server",
            location="opc.tcp://localhost:4841/freeopcua/server/",
            rack=rack_key,
        )


@dataclass(frozen=True)
class SimulatorConfig:
    """
    Configuration for a simulator server.
    Combines server startup callback with a reference to a device from KnownDevices.
    """

    server_setup: Callable[[], ForkProcess]
    startup_delay_seconds: float
    device_factory: Callable[[int], SynnaxDevice]
    device_name: str


def start_modbus_server() -> ForkProcess:
    """Start the Modbus TCP simulator server in a separate process."""
    process = mp_ctx.Process(target=lambda: _run_server(run_modbus_server), daemon=True)
    process.start()
    return process


def start_opcua_server() -> ForkProcess:
    """Start the OPC UA simulator server in a separate process."""
    process = mp_ctx.Process(target=lambda: _run_server(run_opcua_server), daemon=True)
    process.start()
    return process


class Simulator:
    """Available simulator servers for driver testing."""

    MODBUS = SimulatorConfig(
        server_setup=start_modbus_server,
        startup_delay_seconds=2.0,
        device_factory=KnownDevices.modbus_sim,
        device_name="Modbus TCP Test Server",
    )

    OPCUA = SimulatorConfig(
        server_setup=start_opcua_server,
        startup_delay_seconds=2.0,
        device_factory=KnownDevices.opcua_sim,
        device_name="OPC UA Test Server",
    )
