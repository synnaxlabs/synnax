#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.modbus import READ_TYPE_NAMES, ModbusRead
from tests.console.task.modbus_case import ModbusCase

# One channel per Modbus type, as (config type, address).
CHANNELS = [
    ("input_register", 0),
    ("holding_register", 1),
    ("discrete_input", 0),
    ("coil", 1),
]
LISTED = [(READ_TYPE_NAMES[t], a) for t, a in CHANNELS]


class ModbusReadTask(ModbusCase):
    """Configure, start, and stop a Modbus read task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_channels(page)
        self.test_set_rates(page)
        self.test_deploy(page)
        self.test_data_flow(page)
        self.test_reopen_config(page, [])
        self.assert_rows(page, LISTED)

    def test_create_task(self) -> ModbusRead:
        """A new task is not deployed; select the server."""
        self.log("Testing: Create Modbus read task")
        page = self.create_page(ModbusRead, "Modbus Console Read")
        self.assert_not_deployed(page, "new task")
        page.select_server(self.device_name)
        return page

    def test_add_channels(self, page: ModbusRead) -> None:
        """One channel of each type is listed."""
        self.log("Testing: Add channels")
        for config_type, address in CHANNELS:
            page.add_channel(READ_TYPE_NAMES[config_type], address)
        self.assert_rows(page, LISTED)

    def test_data_flow(self, page: ModbusRead) -> None:
        """Deploy created one channel per row; each collects samples at the rate."""
        self.log("Testing: Data flows on the created channels")
        task = self.retrieve_task(page.page_name)
        registers = self.registers(task)
        assert registers == set(CHANNELS), (
            f"Task channels should be {set(CHANNELS)}, got {registers}"
        )
        keys = self.channel_keys(task)
        assert len(keys) == len(CHANNELS), f"Every channel needs a key, got {keys}"
        self.assert_sample_count(page, keys)
