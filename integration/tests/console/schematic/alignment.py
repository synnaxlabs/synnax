#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase
from console.schematic.schematic import Schematic

CHANNEL_NAME = "command_channel"
INDEX_NAME = "idx_channel"


class Alignment(ConsoleCase):
    """
    Test the alignment of symbols in the schematic
    """

    def run(self) -> None:
        console = self.console
        client = self.client

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

        schematic = Schematic(client, console, "set_output_schematic")

        # Set up symbols
        setpoint_symbol = schematic.create_setpoint(CHANNEL_NAME)
        setpoint_symbol.move(-150, 0)
        value_symbol = schematic.create_value(CHANNEL_NAME)
        value_symbol.move(150, 20)
        button_symbol = schematic.create_button(CHANNEL_NAME)
        button_symbol.move(0, -20)
        valve_symbol = schematic.create_valve(CHANNEL_NAME, no_state_channel=True)
        valve_symbol.move(0, 30)

        symbols = [setpoint_symbol, value_symbol, button_symbol, valve_symbol]

        self.log("Align Vertical")
        schematic.align(symbols, "vertical")

        self.log("Distribute Horizontal")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Horizontal")
        button_symbol.move(0, -100)
        setpoint_symbol.move(0, 100)
        schematic.align(symbols, "horizontal")

        self.log("Distribute Vertical")
        schematic.distribute(symbols, "vertical")

        self.log("Align Left")
        schematic.align(symbols, "left")

        self.log("Align Right")
        schematic.align(symbols, "right")

        self.log("Align Top")
        setpoint_symbol.move(-150, 0)
        value_symbol.move(150, 0)
        schematic.align(symbols, "top")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Bottom")
        button_symbol.move(0, -20)
        setpoint_symbol.move(0, 30)
        schematic.align(symbols, "bottom")

        schematic.screenshot()
