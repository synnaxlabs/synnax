#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase
from console.schematic import Setpoint, Value
from console.schematic.schematic import Schematic
from framework.utils import get_random_name


def assert_setpoint(
    schematic: Schematic,
    channel_name: str,
    expected_value: float,
    poll_interval: float = 0.1,
) -> None:
    """Assert that the setpoint value in the Core matches expected.

    Retries until the value matches or timeout is reached.
    """
    elapsed = 0.0
    while elapsed < 3.0:
        actual_value = schematic.get_value(channel_name)
        if actual_value == expected_value:
            return
        sy.sleep(poll_interval)
        elapsed += poll_interval

    actual_value = schematic.get_value(channel_name)
    assert actual_value == expected_value, (
        f"Setpoint value mismatch after 3.0s!\n"
        f"Actual: {actual_value}\nExpected: {expected_value}"
    )


class SetOutput(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        console = self.console
        client = self.client
        CHANNEL_NAME = f"command_channel_{get_random_name()}"
        INDEX_NAME = f"idx_channel_{get_random_name()}"
        self.log("Creating channels")

        index_ch = client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        cmd_ch = client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        self.log("Creating schematic symbols")
        schematic = Schematic(console, "set_output_schematic")

        setpoint_symbol = schematic.create_symbol(
            Setpoint(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )
        setpoint_symbol.move(delta_x=-200, delta_y=0)

        value_symbol = schematic.create_symbol(
            Value(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )
        value_symbol.move(delta_x=200, delta_y=0)

        schematic.connect_symbols(setpoint_symbol, "right", value_symbol, "left")

        set_p_value = 47.23
        setpoint_symbol.set_value(set_p_value)
        self.log(f"Verifying setpoint value: {set_p_value}")
        assert_setpoint(schematic, CHANNEL_NAME, set_p_value)

        set_p_value = 1.0101
        setpoint_symbol.set_value(set_p_value)
        self.log(f"Verifying setpoint value: {set_p_value}")
        assert_setpoint(schematic, CHANNEL_NAME, set_p_value)

        schematic.screenshot()
