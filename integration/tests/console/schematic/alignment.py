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

    def setup(self) -> None:

        super().setup()

        index_ch = self.client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        cmd_ch = self.client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

    def run(self) -> None:
        console = self.console
        client = self.client

        schematic = Schematic(client, console, "set_output_schematic")

        # Set up Symbols
        valve_threeway = schematic.create.valve_threeway(CHANNEL_NAME)
        valve_threeway.move(-150, 0)

        valve_threeway_ball = schematic.create.valve_threeway_ball(CHANNEL_NAME)
        valve_threeway_ball.move(150, -20)

        valve = schematic.create.valve(CHANNEL_NAME)
        valve.move(0, 50)

        setpoint = schematic.create.setpoint(CHANNEL_NAME)
        setpoint.move(-210, 0)

        symbols = [setpoint, valve_threeway, valve_threeway_ball, valve]

        self.log("Align Vertical")
        schematic.align(symbols, "vertical")

        self.log("Distribute Horizontal")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Horizontal")
        valve_threeway.move(0, -100)
        valve_threeway_ball.move(0, 100)
        schematic.align(symbols, "horizontal")

        self.log("Distribute Vertical")
        schematic.distribute(symbols, "vertical")

        self.log("Align Left")
        schematic.align(symbols, "left")

        self.log("Align Right")
        schematic.align(symbols, "right")

        self.log("Align Top")
        valve_threeway.move(-150, 0)
        valve_threeway_ball.move(150, 0)
        schematic.align(symbols, "top")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Bottom")
        valve.move(0, -20)
        valve_threeway.move(0, 30)
        schematic.align(symbols, "bottom")

        self.log("Rotate Individual Clockwise")
        schematic.rotate(symbols, "clockwise", group=False)

        self.log("Rotate Individual Counter-Clockwise")
        schematic.rotate(symbols, "counterclockwise", group=False)

        self.log("Rotate Group Clockwise")
        schematic.rotate(symbols, "clockwise", group=True)

        self.log("Rotate Group Counter-Clockwise")
        schematic.rotate(symbols, "counterclockwise", group=True)

        schematic.screenshot()
