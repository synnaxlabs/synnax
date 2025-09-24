#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from test.console.console_case import ConsoleCase

import synnax as sy


class Schematic_Set_Output(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        console = self.console
        console.schematic.new()

        CHANNEL_NAME = "command_channel"
        INDEX_NAME = "idx_channel"

        console.channels.create(
            INDEX_NAME,
            is_index=True,
        )
        console.channels.create(
            CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=INDEX_NAME,
        )

        setpoint_symbol = console.schematic.create_setpoint(CHANNEL_NAME)
        setpoint_symbol.move(-200, 0)

        value_symbol = console.schematic.create_value(CHANNEL_NAME)
        value_symbol.move(200, 0)

        console.schematic.connect_symbols(
            setpoint_symbol, "right", value_symbol, "left"
        )

        set_p_value = 47.23
        self._log_message(f"Verifying setpoint value: {set_p_value}")
        setpoint_symbol.set_value(set_p_value)
        actual_value = self.get_value(CHANNEL_NAME)

        assert (
            actual_value == set_p_value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {set_p_value}"

        set_p_value = 1.0101
        self._log_message(f"Verifying setpoint value: {set_p_value}")
        setpoint_symbol.set_value(set_p_value)
        actual_value = self.get_value(CHANNEL_NAME)

        assert (
            actual_value == set_p_value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {set_p_value}"

        console.schematic.screenshot()
