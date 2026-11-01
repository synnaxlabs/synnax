#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from examples.modbus import ModbusGatewaySim

import synnax as sy
from tests.console.device.connect_network import NETWORK
from tests.console.task.modbus_case import ModbusCase
from x import random_name


class ConnectGateway(ModbusCase):
    """Connect a unit behind a Modbus gateway with the unit, byte order, and timing
    settings of the connect form. The Driver checks the unit before the Console saves
    the device, then counts its requests in the diagnostics tab."""

    sim_classes = [ModbusGatewaySim]

    def run(self) -> None:
        devices = self.console.devices
        devices.open_connect()
        devices.pick_integration(NETWORK)
        name = f"Gateway unit {random_name()}"
        self.test_silent_unit(name)
        self.test_connect_unit(name)
        self.test_diagnostics(name)

    def fill_unit(self, unit_id: int) -> None:
        modal = self.console.devices.modal
        modal.get_by_role("tab", name="Unit", exact=True).click(timeout=5000)
        modal.get_by_label(re.compile(r"^Unit ID\b")).fill(str(unit_id))

    def test_silent_unit(self, name: str) -> None:
        """A unit the gateway does not answer for blocks the connection."""
        self.log("Testing: Connect a unit that does not answer")
        devices = self.console.devices
        self.fill_unit(9)
        devices.fill_connect(
            name=name, host=ModbusGatewaySim.host, port=ModbusGatewaySim.port
        )
        devices.modal.get_by_text(re.compile("Failed to connect")).wait_for(
            state="visible", timeout=15000
        )
        assert self.client.devices.retrieve(name=name, ignore_not_found=True) is None, (
            f"'{name}' should not exist after a failed connection"
        )

    def test_connect_unit(self, name: str) -> None:
        """Unit 2 answers, and the device keeps the unit, byte order, and timeout."""
        self.log("Testing: Connect unit 2 with its settings")
        devices = self.console.devices
        layout = self.console.layout
        self.fill_unit(2)
        layout.click_btn("Byte order")
        layout.select_from_dropdown("Big-endian (ABCD)", exact=True)
        devices.modal.get_by_role("tab", name="Advanced", exact=True).click()
        devices.modal.get_by_label(re.compile(r"^Timeout\b")).fill("300")
        devices.fill_connect(
            name=name, host=ModbusGatewaySim.host, port=ModbusGatewaySim.port
        )
        devices.modal.wait_for(state="hidden", timeout=15000)
        dev = self.client.devices.retrieve(name=name)
        self.track_test_devices([dev])
        connection = sy.device.ensure_object(dev.properties, "connection")
        expected = {
            "unit_id": 2,
            "swap_bytes": False,
            "swap_words": True,
            "timeout_ms": 300,
        }
        stored = {k: connection.get(k) for k in expected}
        assert stored == expected, f"'{name}' should store {expected}, got {stored}"

    def test_diagnostics(self, name: str) -> None:
        """The diagnostics tab shows the requests the Driver counted for the unit."""
        self.log("Testing: Diagnostics show the request counts")
        self.wait_for_stats(name)
        devices = self.console.devices
        layout = self.console.layout
        devices.ctx_menu.action(devices.get(name), "Edit connection")
        devices.modal.wait_for(state="visible", timeout=5000)
        devices.modal.get_by_role("tab", name="Diagnostics", exact=True).click(
            timeout=5000
        )
        shown = {
            label: devices.modal.locator(f"dt:text-is('{label}') + dd").inner_text()
            for label in ("Requests", "Timeouts", "Exceptions", "Last response")
        }
        assert int(shown["Requests"]) > 0, f"Requests should be counted, got {shown}"
        assert shown["Timeouts"] == "0" and shown["Exceptions"] == "0", shown
        assert re.fullmatch(r"\d+\.\d ms", shown["Last response"]), shown
        layout.close_modal(layout.MODAL_SELECTOR)

    def wait_for_stats(self, name: str) -> None:
        """Wait for the Driver to report the counters of a device in its status."""
        timer = sy.Timer()
        while timer.elapsed() < 15 * sy.TimeSpan.SECOND:
            dev = self.client.devices.retrieve(name=name, include_status=True)
            details = None if dev.status is None else dev.status.details
            if details is not None and details.data is not None:
                return
            sy.sleep(0.5)
        raise AssertionError(f"'{name}' should report its request counters")
