#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import sys

from examples.modbus import ModbusSerialSim

import synnax as sy
from tests.console.device.connect_network import NETWORK
from tests.console.task.modbus_case import ModbusCase
from x import random_name


class ConnectSerial(ModbusCase):
    """Connect a unit on a serial line through the connect form. The Driver checks the
    port and the unit before the Console saves the device. The line is a
    pseudo-terminal, so the case runs only on POSIX."""

    sim_classes = [ModbusSerialSim]

    def setup(self) -> None:
        if sys.platform == "win32":
            self.skip("Pseudo-terminals need POSIX")
        super().setup()

    def run(self) -> None:
        devices = self.console.devices
        devices.open_connect()
        devices.pick_integration(NETWORK)
        devices.modal.get_by_role("button", name="Serial", exact=True).click()
        name = f"Serial unit {random_name()}"
        self.test_missing_port(name)
        self.test_connect_unit(name)

    def test_missing_port(self, name: str) -> None:
        """A port that does not exist blocks the connection."""
        self.log("Testing: Connect a missing serial port")
        devices = self.console.devices
        devices.fill_connect_serial(name=name, serial_port="/dev/synnax-missing-port")
        devices.modal.get_by_text(re.compile("Failed to connect")).wait_for(
            state="visible", timeout=15000
        )
        assert self.client.devices.retrieve(name=name, ignore_not_found=True) is None, (
            f"'{name}' should not exist after a failed connection"
        )

    def test_connect_unit(self, name: str) -> None:
        """Unit 2 answers on the line, and the device keeps the line settings."""
        self.log("Testing: Connect unit 2 on the serial line")
        devices = self.console.devices
        layout = self.console.layout
        layout.click_btn("Parity")
        layout.select_from_dropdown("None", exact=True)
        devices.modal.get_by_role("tab", name="Unit", exact=True).click(timeout=5000)
        devices.modal.get_by_label(re.compile(r"^Unit ID\b")).fill("2")
        devices.fill_connect_serial(name=name, serial_port=ModbusSerialSim.serial_port)
        devices.modal.wait_for(state="hidden", timeout=15000)
        dev = self.client.devices.retrieve(name=name)
        self.track_test_devices([dev])
        connection = sy.device.ensure_object(dev.properties, "connection")
        expected = {
            "transport": "rtu",
            "serial_port": ModbusSerialSim.serial_port,
            "unit_id": 2,
            "baud_rate": 19200,
            "parity": "none",
            "stop_bits": 1,
        }
        stored = {k: connection.get(k) for k in expected}
        assert stored == expected, f"'{name}' should store {expected}, got {stored}"
