#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from test.console.schematic import Schematic


class Schematic_Set_Output(Schematic):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        CHANNEL_NAME = "command_channel"

        self.create_a_channel("index_channel", is_index=True)
        self.create_a_channel(
            CHANNEL_NAME, is_index=False, data_type="Float64", index="index_channel"
        )

        setpoint_node = self.add_to_schematic("Setpoint", CHANNEL_NAME)
        setpoint_node.move(-200, 0)

        value_node = self.add_to_schematic("Value", CHANNEL_NAME)
        value_node.move(200, 0)

        set_p_value = 47.23333333
        setpoint_node.set_value(set_p_value)
        time.sleep(0.1)
        actual_value = self.get_value(CHANNEL_NAME)

        self._log_message(f"Verifying setpoint value: {set_p_value}")
        assert (
            actual_value == set_p_value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {set_p_value}"

        set_p_value = 1.0101
        setpoint_node.set_value(set_p_value)
        time.sleep(0.2)
        actual_value = self.get_value(CHANNEL_NAME)

        self._log_message(f"Verifying setpoint value: {set_p_value}")
        assert (
            actual_value == set_p_value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {set_p_value}"
