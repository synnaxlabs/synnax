#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus task pages for Console UI automation."""

from playwright.sync_api import Locator

from console.layout import LayoutClient
from console.task_page import TaskPage

SERVER_FIELD_LABEL = "Modbus server"
CHANNEL_FIELDS_SELECTOR = ".console-channel-item"

# Config channel type to the name shown in the channel type dropdown.
READ_TYPE_NAMES = {
    "coil": "Coil",
    "discrete_input": "Discrete",
    "holding_register": "Holding register",
    "input_register": "Register",
}
WRITE_TYPE_NAMES = {"coil": "Coil", "holding_register": "Holding register"}


class ModbusTask(TaskPage):
    """Shared Modbus task page. Each channel is a row of type, address, and, for
    registers, data type."""

    def _rows(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def _fields(self, index: int) -> Locator:
        return self._rows().nth(index).locator(CHANNEL_FIELDS_SELECTOR)

    def select_server(self, name: str) -> None:
        """Pick the Modbus server the task talks to. Selecting a server that is not
        configured yet opens the connect modal prefilled; confirm it.

        :param name: Name of the server device.
        """
        configured = self.client.devices.retrieve(name=name).configured
        self.select_device(SERVER_FIELD_LABEL, name)
        if configured:
            return
        modal = self.page.locator(LayoutClient.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        # Toasts stack over the modal footer and swallow the click.
        self.notifications.close_all()
        modal.get_by_role("button", name="Connect", exact=True).click()
        modal.wait_for(state="hidden", timeout=15000)

    def add_channel(
        self, type_name: str, address: int, data_type: str | None = None
    ) -> None:
        """Append a channel row and fill it in.

        :param type_name: Type as shown in the type dropdown, e.g. "Register".
        :param address: Modbus address.
        :param data_type: Register data type, e.g. "float32". Registers only.
        """
        index = self._rows().count()
        self.add_channel_row(index)
        self._rows().nth(index).wait_for(state="visible", timeout=5000)
        self.set_channel_type(index, type_name)
        self.set_address(index, address)
        if data_type is not None:
            self._fields(index).get_by_role("button").nth(1).click()
            self.layout.select_from_dropdown(data_type, exact=True)

    def set_channel_type(self, index: int, type_name: str) -> None:
        """Set the type of the channel row at ``index``."""
        self._fields(index).get_by_role("button").first.click()
        self.layout.select_from_dropdown(type_name, exact=True)

    def set_address(self, index: int, address: int) -> None:
        """Set the address of the channel row at ``index``."""
        self._fields(index).locator("input").first.fill(str(address))

    def channels(self) -> list[tuple[str, int]]:
        """Return the (type name, address) of each listed channel, in order."""
        result: list[tuple[str, int]] = []
        for i in range(self._rows().count()):
            fields = self._fields(i)
            type_name = fields.get_by_role("button").first.inner_text().strip()
            address = int(fields.locator("input").first.input_value())
            result.append((type_name, address))
        return result


class ModbusRead(ModbusTask):
    """Modbus read task page."""

    page_type = "Modbus read task"
    pluto_label: str = ".console-task-configure--modbus_read"


class ModbusWrite(ModbusTask):
    """Modbus write task page."""

    page_type = "Modbus write task"
    pluto_label: str = ".console-task-configure--modbus_write"
