#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.modbus import WRITE_TYPE_NAMES, ModbusWrite
from tests.console.task.modbus_case import ModbusCase

# Coils and float32 holding registers, as (config type, address, data type).
CHANNELS = [
    ("coil", 20, None),
    ("coil", 21, None),
    ("holding_register", 20, "float32"),
    ("holding_register", 22, "float32"),
]
LISTED = [(WRITE_TYPE_NAMES[t], a) for t, a, _ in CHANNELS]


class ModbusWriteTask(ModbusCase):
    """Configure, start, and stop a Modbus write task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_channels(page)
        self.test_deploy(page)
        self.test_send_commands(page)
        self.test_stop(page)
        self.test_reopen_config(page, [])
        self.assert_rows(page, LISTED)

    def test_create_task(self) -> ModbusWrite:
        """Create the task page and select the server."""
        self.log("Testing: Create Modbus write task")
        page = self.create_page(ModbusWrite, "Modbus Console Write")
        page.select_server(self.device_name)
        return page

    def test_add_channels(self, page: ModbusWrite) -> None:
        """Coil and holding register command channels are listed."""
        self.log("Testing: Add command channels")
        for config_type, address, data_type in CHANNELS:
            page.add_channel(WRITE_TYPE_NAMES[config_type], address, data_type)
        self.assert_rows(page, LISTED)

    def test_send_commands(self, page: ModbusWrite) -> None:
        """Deploy created one command channel per row; commands reach the server."""
        self.log("Testing: Send commands")
        task = self.retrieve_task(page.page_name)
        registers = self.registers(task)
        expected = {(t, a) for t, a, _ in CHANNELS}
        assert registers == expected, (
            f"Task channels should be {expected}, got {registers}"
        )
        keys = self.channel_keys(task)
        assert len(keys) == len(CHANNELS), (
            f"Every command channel needs a key, got {keys}"
        )
        self.send_commands(page, task)
