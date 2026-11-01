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
from console.page import PageType
from console.task_page import TaskPage

SERVER_FIELD_LABEL = "Modbus server"

# Config channel type to the name shown in the channel type dropdown.
READ_TYPE_NAMES = {
    "coil": "Coil",
    "discrete_input": "Discrete input",
    "holding_register": "Holding register",
    "input_register": "Input register",
    "holding_register_bit": "Holding register bit",
    "input_register_bit": "Input register bit",
}
WRITE_TYPE_NAMES = {
    "coil": "Coil",
    "holding_register": "Holding register",
    "holding_register_bit": "Holding register bit",
}


class ModbusTask(TaskPage):
    """Shared Modbus task page. Each channel is a row of type, address, and, for
    registers, data type."""

    # The element that holds a row's fields.
    fields_selector: str

    def _rows(self) -> Locator:
        return self._channel_list().get_by_role("option")

    def _fields(self, index: int) -> Locator:
        return self._rows().nth(index).locator(self.fields_selector)

    def select_server(self, name: str) -> None:
        """Pick the Modbus server the task talks to. Selecting a server that is not
        configured yet opens the connect modal prefilled; confirm it.

        :param name: Name of the server device.
        """
        configured = self.client.devices.retrieve(name=name).configured
        self.select_device(SERVER_FIELD_LABEL, name=name)
        if configured:
            return
        modal = self.page.locator(LayoutClient.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        # Toasts stack over the modal footer and swallow the click.
        self.notifications.close_all()
        modal.get_by_role("button", name="Connect", exact=True).click()
        modal.wait_for(state="hidden", timeout=15000)

    def add_channel(
        self, type_name: str, *, address: int | str, data_type: str | None = None
    ) -> None:
        """Fill in the next channel row.

        :param type_name: Type as shown in the type dropdown, e.g. "Coil".
        :param address: Modbus address, or a six-digit Modicon address such as
            "400001".
        :param data_type: Register data type, e.g. "float32". Registers only.
        """
        index = self.next_channel_row()
        self.set_channel_type(index, type_name)
        self.set_address(index, address)
        if data_type is not None:
            self.set_data_type(index, data_type)

    def set_channel_type(self, index: int, type_name: str) -> None:
        """Set the type of the channel row at ``index``."""
        self._fields(index).get_by_role("button").first.click()
        self.layout.select_from_dropdown(type_name, exact=True)

    def set_data_type(self, index: int, data_type: str) -> None:
        """Set the data type of the register channel row at ``index``."""
        self._fields(index).get_by_role("button").nth(1).click()
        self.layout.select_from_dropdown(data_type, exact=True)

    def set_address(self, index: int, address: int | str) -> None:
        """Set the address of the channel row at ``index``."""
        field = self._fields(index).locator("input").first
        field.fill(str(address))
        field.press("Enter")

    def set_bit(self, index: int, bit: int) -> None:
        """Set the bit of the register bit channel row at ``index``."""
        self._fill(index, "Bit", bit)

    def set_string_length(self, index: int, length: int) -> None:
        """Set the string length of the string channel row at ``index``."""
        self._fill(index, "String length", length)

    def _fill(self, index: int, label: str, value: int) -> None:
        field = self._fields(index).get_by_label(label, exact=True)
        field.fill(str(value))
        field.press("Enter")

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

    page_type: PageType = "Modbus read task"
    pluto_label: str = ".console-task-configure--modbus_read"
    fields_selector = ".lyra-list__cells"


class ModbusWrite(ModbusTask):
    """Modbus write task page."""

    page_type: PageType = "Modbus write task"
    pluto_label: str = ".console-task-configure--modbus_write"
    fields_selector = ".console-channel-item"
