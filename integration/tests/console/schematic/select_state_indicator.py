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
from console.schematic import Select, StateIndicator
from framework.utils import get_random_name


class SelectStateIndicator(ConsoleCase):
    """Test Select and State Indicator symbols together.

    Creates a Select (write) and State Indicator (read) pointing at the same
    channel, sends a value via Select, and verifies both the channel value and
    the State Indicator label update correctly.
    """

    def run(self) -> None:
        console = self.console
        client = self.client
        CHANNEL_NAME = f"state_channel_{get_random_name()}"
        INDEX_NAME = f"idx_channel_{get_random_name()}"

        self.log("Creating channels")
        index_ch = client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        state_ch = client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        options = [
            {"name": "Open", "value": 1},
            {"name": "Closed", "value": 2},
            {"name": "Standby", "value": 3},
        ]

        self.log("Creating schematic")
        schematic = console.workspace.create_schematic("select_state_indicator")
        self._cleanup_pages.append(schematic.page_name)

        self.log("Creating Select symbol")
        select_symbol = schematic.create_symbol(
            Select(
                label="State Control",
                channel_name=CHANNEL_NAME,
                options=options,
            )
        )
        select_symbol.move(delta_x=-200, delta_y=0)

        self.log("Creating State Indicator symbol")
        indicator_symbol = schematic.create_symbol(
            StateIndicator(
                label="State Display",
                channel_name=CHANNEL_NAME,
                options=options,
            )
        )
        indicator_symbol.move(delta_x=200, delta_y=0)

        schematic.connect_symbols(select_symbol, "right", indicator_symbol, "left")

        self.log("Sending 'Open' via Select")
        select_symbol.send("Open")
        self.wait_for_eq(CHANNEL_NAME, 1)
        self._assert_indicator_label(indicator_symbol, "Open")

        self.log("Sending 'Closed' via Select")
        select_symbol.send("Closed")
        self.wait_for_eq(CHANNEL_NAME, 2)
        self._assert_indicator_label(indicator_symbol, "Closed")

        self.log("Sending 'Standby' via Select")
        select_symbol.send("Standby")
        self.wait_for_eq(CHANNEL_NAME, 3)
        self._assert_indicator_label(indicator_symbol, "Standby")

        schematic.screenshot()

    def _assert_indicator_label(
        self,
        indicator: StateIndicator,
        expected: str,
        timeout: float = 5.0,
    ) -> None:
        elapsed = 0.0
        poll = 0.25
        while elapsed < timeout:
            actual = indicator.get_label()
            if actual == expected:
                return
            sy.sleep(poll)
            elapsed += poll
        actual = indicator.get_label()
        assert actual == expected, (
            f"State Indicator label mismatch after {timeout}s: "
            f"expected '{expected}', got '{actual}'"
        )
