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
from tests.driver.modbus_task import (
    assert_sim_coils,
    assert_sim_registers,
    float32_words,
)

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
        self.test_change_data_type(page)
        self.test_reopen_config(page, [])
        self.assert_rows(page, LISTED)
        self.test_register_bit(page)

    def test_create_task(self) -> ModbusWrite:
        """Create the task page and select the server."""
        self.log("Testing: Create Modbus write task")
        page = self.create_page(ModbusWrite, prefix="Modbus Console Write")
        page.select_server(self.device_name)
        return page

    def test_add_channels(self, page: ModbusWrite) -> None:
        """Coil and holding register command channels are listed."""
        self.log("Testing: Add command channels")
        for config_type, address, data_type in CHANNELS:
            page.add_channel(
                WRITE_TYPE_NAMES[config_type], address=address, data_type=data_type
            )
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
        self.send_commands(page, task=task)
        # The last round of commands is 100 to 103.
        assert_sim_coils(20, [True, True])
        assert_sim_registers(20, [*float32_words(102), *float32_words(103)])

    def test_change_data_type(self, page: ModbusWrite) -> None:
        """A data type change gives the register a new command channel, and a change
        back gives it the old channel again."""
        self.log("Testing: Change a command data type")
        before = self.channel_keys(self.retrieve_task(page.page_name))
        page.set_data_type(3, "uint16")
        self.test_deploy(page)
        self.stop(page)
        changed = self.channel_keys(self.retrieve_task(page.page_name))
        assert changed[:3] == before[:3] and changed[3] != before[3], (
            f"Only the last command channel should change, got {before} -> {changed}"
        )
        name = self.client.channels.retrieve(changed[3]).name
        assert name.endswith("_holding_register_output_22_u16_cmd"), name
        page.set_data_type(3, "float32")
        self.test_deploy(page)
        self.stop(page)
        after = self.channel_keys(self.retrieve_task(page.page_name))
        assert after == before, f"Command channels should be {before}, got {after}"

    def test_register_bit(self, page: ModbusWrite) -> None:
        """A register bit takes a Modicon address and changes only its bit."""
        self.log("Testing: Write a register bit at a Modicon address")
        index = len(CHANNELS)
        page.add_channel(WRITE_TYPE_NAMES["holding_register_bit"], address="400141")
        page.set_bit(index, 4)
        self.test_deploy(page)
        task = self.retrieve_task(page.page_name)
        bit = self.config_channels(task)[index]
        stored = (bit["type"], bit["address"], bit["bit"])
        expected = ("holding_register_bit", 140, 4)
        assert stored == expected, f"The bit should be {expected}, got {stored}"
        name = self.client.channels.retrieve(self.channel_keys(task)[index]).name
        assert name.endswith("_holding_register_bit_output_140_4_cmd"), name
        self.send_commands(page, task=task)
        # Register 140 holds 0b10001100.
        assert_sim_registers(140, [0b10011100])
        self.stop(page)
