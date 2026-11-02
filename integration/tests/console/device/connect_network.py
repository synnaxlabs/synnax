#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from examples.modbus import ModbusSim

import synnax as sy
from console.task.modbus import SERVER_FIELD_LABEL, ModbusRead
from tests.console.task.modbus_case import ModbusCase
from x import random_name

NETWORK = "Modbus server"
# Nothing listens here, so the Driver's connection check fails.
CLOSED_PORT = ModbusSim.port + 1


class ConnectNetwork(ModbusCase):
    """Connect Modbus servers through the connect picker and a task form. The Driver
    checks each connection before the Console saves the device."""

    def run(self) -> None:
        self.test_connect_from_toolbar()
        self.test_connect_from_task_form()

    def _retrieve_connected(self, name: str) -> sy.Device:
        """Retrieve a device the Console connected, and track it for teardown."""
        dev = self.client.devices.retrieve(name=name)
        self.track_test_devices([dev])
        assert dev.configured, f"'{name}' should be configured"
        location = f"{ModbusSim.host}:{ModbusSim.port}"
        assert dev.location == location, (
            f"'{name}' should be at {location}, got {dev.location}"
        )
        return dev

    def test_connect_from_toolbar(self) -> None:
        """The picker opens the form on the only Driver that runs Modbus. A server
        that does not answer blocks the connection, and one that answers connects."""
        self.log("Testing: Connect a server from the toolbar")
        devices = self.console.devices
        devices.open_connect()
        devices.pick_integration(NETWORK)
        connect_from = self.console.layout.get_dropdown_value("Connect from")
        rack_name = self.options.driver_rack
        assert connect_from == rack_name, (
            f"The form should connect from '{rack_name}', got '{connect_from}'"
        )

        name = f"Toolbar server {random_name()}"
        devices.fill_connect(name=name, host=ModbusSim.host, port=CLOSED_PORT)
        devices.modal.get_by_text(re.compile("Failed to connect")).wait_for(
            state="visible", timeout=15000
        )
        assert self.client.devices.retrieve(name=name, ignore_not_found=True) is None, (
            f"'{name}' should not exist after a failed connection"
        )

        devices.fill_connect(name=name, host=ModbusSim.host, port=ModbusSim.port)
        devices.modal.wait_for(state="hidden", timeout=15000)
        devices.get(name)
        dev = self._retrieve_connected(name)
        rack = self.client.racks.retrieve(name=self.options.driver_rack)
        assert dev.rack == rack.key, (
            f"'{name}' should be on '{self.options.driver_rack}'"
        )

    def test_connect_from_task_form(self) -> None:
        """The "Connect Modbus server" action of the server dropdown opens the form,
        and the connected server becomes the selected server."""
        self.log("Testing: Connect a server from a task form")
        self.create_page(ModbusRead, prefix="Connect Read")
        layout = self.console.layout
        layout.click_btn(SERVER_FIELD_LABEL)
        layout.dialog.get_by_text(f"Connect {NETWORK}", exact=True).click(timeout=5000)
        devices = self.console.devices
        devices.modal.wait_for(state="visible", timeout=5000)
        name = f"Form server {random_name()}"
        devices.fill_connect(name=name, host=ModbusSim.host, port=ModbusSim.port)
        devices.modal.wait_for(state="hidden", timeout=15000)
        layout.wait_for_dropdown_value(SERVER_FIELD_LABEL, name)
        self._retrieve_connected(name)
