#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus invalid configuration tests through the Console.

The Console form blocks some invalid settings before they reach the Driver and
renders a field error; the rest deploy and the Driver rejects them with an error
status. A bad channel key cannot be entered through the form, and a zero stream rate
is healed by the Core on save; both stay with ``driver/modbus_invalid``.
"""

from console.task.modbus import READ_TYPE_NAMES, SERVER_FIELD_LABEL, ModbusRead
from tests.console.task.modbus_case import ModbusCase

REGISTER = READ_TYPE_NAMES["input_register"]
BAD_ADDRESS = 65000


class ModbusInvalidConfig(ModbusCase):
    """Verify invalid Modbus read task configurations are rejected.

    Tests (run sequentially on one task page):
        1. No device selected: the form blocks the deploy.
        2. Invalid rates: stream rate above sample rate, the form blocks the deploy.
        3. Invalid address: a register the simulator does not serve, the Driver
           rejects the start.
        4. Duplicate channel: a second task on the same register is rejected while
           the first one runs.
    """

    def run(self) -> None:
        page = self.test_no_device()
        self.test_invalid_rates(page)
        self.test_invalid_address(page)
        self.test_duplicate_channel(page)

    def test_no_device(self) -> ModbusRead:
        self.log("Testing: No device selected")
        page = self.create_page(ModbusRead, "Modbus Console Invalid")
        self.assert_device_required(page, SERVER_FIELD_LABEL)
        return page

    def test_invalid_rates(self, page: ModbusRead) -> None:
        self.log("Testing: Invalid rates (sample_rate < stream_rate)")
        page.select_server(self.device_name)
        page.add_channel(REGISTER, 0)
        self.assert_rates_rejected(page)

    def test_invalid_address(self, page: ModbusRead) -> None:
        self.log("Testing: Invalid register address (runtime)")
        self.set_rates(page)
        page.set_address(0, BAD_ADDRESS)
        self.assert_driver_rejects(page, "invalid address")

    def test_duplicate_channel(self, page: ModbusRead) -> None:
        """Run the task, then deploy a second task on the same register."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        page.set_address(0, 0)
        page.deploy()
        # Page objects locate their pane by task type, so the first tab closes before a
        # second read task opens.
        self.console.close_all_tabs()
        second = self.create_page(ModbusRead, "Modbus Console Duplicate")
        second.select_server(self.device_name)
        second.add_channel(REGISTER, 0)
        self.assert_driver_rejects(second, "duplicate channel")
        self.console.tasks.stop(page.page_name)
