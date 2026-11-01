#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.modbus.server import STRING_START

from console.task.modbus import READ_TYPE_NAMES, ModbusRead
from tests.console.task.modbus_case import ModbusCase
from x.telem import to_int

# One channel per Modbus type, as (config type, address).
NUMBERS = [
    ("input_register", 0),
    ("holding_register", 1),
    ("discrete_input", 0),
    ("coil", 1),
]
STRING = ("input_register", STRING_START)
STRING_VALUE = "SYNNAX"
# Input register 140 holds 0b10001100, so bit 0 is off and bit 2 is on.
BIT = ("input_register_bit", 140)
BIT_INDEX = 2
CHANNELS = [*NUMBERS, STRING, BIT]
LISTED = [(READ_TYPE_NAMES[t], a) for t, a in CHANNELS]


class ModbusReadTask(ModbusCase):
    """Configure, start, and stop a Modbus read task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_channels(page)
        self.test_set_rates(page)
        self.test_deploy(page)
        self.test_data_flow(page)
        self.test_string_and_bit(page)
        self.test_reopen_config(page, [])
        self.assert_rows(page, LISTED)

    def test_create_task(self) -> ModbusRead:
        """A new task is not deployed; select the server."""
        self.log("Testing: Create Modbus read task")
        page = self.create_page(ModbusRead, prefix="Modbus Console Read")
        self.assert_not_deployed(page, label="new task")
        page.select_server(self.device_name)
        return page

    def test_add_channels(self, page: ModbusRead) -> None:
        """One channel of each type, a string, and a register bit are listed."""
        self.log("Testing: Add channels")
        for config_type, address in NUMBERS:
            page.add_channel(READ_TYPE_NAMES[config_type], address=address)
        page.add_channel(
            READ_TYPE_NAMES[STRING[0]], address=STRING[1], data_type="String"
        )
        page.set_string_length(CHANNELS.index(STRING), len(STRING_VALUE))
        page.add_channel(READ_TYPE_NAMES[BIT[0]], address=BIT[1])
        page.set_bit(CHANNELS.index(BIT), BIT_INDEX)
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
        self.assert_sample_count(page, keys=keys)

    def test_string_and_bit(self, page: ModbusRead) -> None:
        """The string channel holds the simulator's string at its length, and the bit
        channel holds the chosen bit."""
        self.log("Testing: String and bit values")
        keys = self.channel_keys(self.retrieve_task(page.page_name))
        string = self.client.read_latest(keys[CHANNELS.index(STRING)])[-1]
        assert string == STRING_VALUE, (
            f"String should be {STRING_VALUE}, got {string!r}"
        )
        bit = to_int(self.client.read_latest(keys[CHANNELS.index(BIT)])[-1])
        assert bit == 1, f"Bit {BIT_INDEX} should be on, got {bit}"
