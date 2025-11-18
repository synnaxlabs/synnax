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

from dataclasses import dataclass
from pathlib import Path
from typing import Callable

from synnax.hardware import modbus, opcua
from synnax.hardware.device import Device as SynnaxDevice


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

    Combines server startup details with a reference to a device from KnownDevices.
    This allows simulators to reference device configurations without duplicating them.
    """

    server_script: Path
    startup_delay_seconds: float
    device_factory: Callable[[int], SynnaxDevice]
    device_name: str
    """The name of the device (for easy retrieval without calling the factory)."""


class Simulator:
    """Available simulator servers for driver testing."""

    MODBUS = SimulatorConfig(
        server_script=Path("client/py/examples/modbus/server.py"),
        startup_delay_seconds=2.0,
        device_factory=KnownDevices.modbus_sim,
        device_name="Modbus TCP Test Server",
    )

    OPCUA = SimulatorConfig(
        server_script=Path("client/py/examples/opcua/server.py"),
        startup_delay_seconds=2.0,
        device_factory=KnownDevices.opcua_sim,
        device_name="OPC UA Test Server",
    )
