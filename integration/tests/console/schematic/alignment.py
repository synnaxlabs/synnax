#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.schematic import (
    Schematic,
    Setpoint,
    Valve,
    ValveThreeWay,
    ValveThreeWayBall,
)

import synnax as sy

CHANNEL_NAME = "alignment_command_channel"
INDEX_NAME = "alignment_idx_channel"


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
        valve_threeway = schematic.create_symbol(
            ValveThreeWay(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_threeway.move(delta_x=-150, delta_y=0)

        valve_threeway_ball = schematic.create_symbol(
            ValveThreeWayBall(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve_threeway_ball.move(delta_x=150, delta_y=-20)

        valve = schematic.create_symbol(
            Valve(
                label=CHANNEL_NAME,
                state_channel=CHANNEL_NAME,
                command_channel=CHANNEL_NAME,
            )
        )
        valve.move(delta_x=0, delta_y=50)

        setpoint = schematic.create_symbol(
            Setpoint(label=CHANNEL_NAME, channel_name=CHANNEL_NAME)
        )
        setpoint.move(delta_x=-210, delta_y=0)

        symbols = [setpoint, valve_threeway, valve_threeway_ball, valve]

        self.log("Align Vertical")
        schematic.align(symbols, "vertical")

        self.log("Distribute Horizontal")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Horizontal")
        valve_threeway.move(delta_x=0, delta_y=-100)
        valve_threeway_ball.move(delta_x=0, delta_y=100)
        schematic.align(symbols, "horizontal")

        self.log("Distribute Vertical")
        schematic.distribute(symbols, "vertical")

        self.log("Align Left")
        schematic.align(symbols, "left")

        self.log("Align Right")
        schematic.align(symbols, "right")

        self.log("Align Top")
        valve_threeway.move(delta_x=-150, delta_y=0)
        valve_threeway_ball.move(delta_x=150, delta_y=0)
        schematic.align(symbols, "top")
        schematic.distribute(symbols, "horizontal")

        self.log("Align Bottom")
        valve.move(delta_x=0, delta_y=-20)
        valve_threeway.move(delta_x=0, delta_y=30)
        schematic.align(symbols, "bottom")

        self.log("Rotate Individual Clockwise")
        schematic.rotate(symbols, "clockwise", group=False)

        self.log("Rotate Individual Counterclockwise")
        schematic.rotate(symbols, "counterclockwise", group=False)

        self.log("Rotate Group Clockwise")
        schematic.rotate(symbols, "clockwise", group=True)

        self.log("Rotate Group Counterclockwise")
        schematic.rotate(symbols, "counterclockwise", group=True)

        schematic.screenshot()
