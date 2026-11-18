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
from console.schematic import Schematic, Select, StateIndicator
from synnax import enumeration
from x import random_name


class SelectStateIndicatorEnumeration(ConsoleCase):
    """Test Select and State Indicator symbols that draw an enumeration.

    Both symbols name one enumeration. The test sends an entry through the Select and
    reads it on the State Indicator, then renames an entry in the Core and checks that
    both symbols draw the new name with no edit to the schematic.
    """

    def run(self) -> None:
        console = self.console
        client = self.client
        CHANNEL_NAME = f"state_channel_{random_name()}"
        INDEX_NAME = f"idx_channel_{random_name()}"

        self.log("Creating channels")
        index_ch = client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.FLOAT64,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        self.log("Creating enumeration")
        valve = client.enumerations.create(
            name=f"valve_state_{random_name()}",
            entries=[
                enumeration.Entry(key="open", value=1, name="Open"),
                enumeration.Entry(key="closed", value=2, name="Closed"),
            ],
        )

        self.log("Creating schematic")
        schematic = console.pages.create(
            Schematic, "select_state_indicator_enumeration"
        )
        self._cleanup_pages.append(schematic.page_name)

        self.log("Creating Select symbol")
        select = Select(
            label="State Control",
            channel_name=CHANNEL_NAME,
            enumeration=valve.name,
        )
        schematic.create_symbol(select)
        select.move(delta_x=-200, delta_y=0)

        self.log("Creating State Indicator symbol")
        indicator = StateIndicator(
            label="State Display",
            channel_name=CHANNEL_NAME,
            enumeration=valve.name,
        )
        schematic.create_symbol(indicator)
        indicator.move(delta_x=200, delta_y=0)

        self.log("Sending 'Open' via Select")
        select.send("Open")
        self.wait_for_eq(CHANNEL_NAME, 1)
        indicator.wait_for_label("Open")

        self.log("Renaming 'Closed' to 'Shut' in the Core")
        client.enumerations.dispatch(
            valve.key,
            enumeration.set(
                enumeration.SetPayload(path=["entries", "closed", "name"], value="Shut")
            ),
        )

        self.log("Sending 'Shut' via Select")
        select.send("Shut")
        self.wait_for_eq(CHANNEL_NAME, 2)
        indicator.wait_for_label("Shut")

        schematic.screenshot()
