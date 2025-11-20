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


class EditSymbolProps(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        schematic = Schematic(self.client, self.console, "edit_symbol_props")

        self.test_value_props(schematic)
        self.test_button_props(schematic)
        self.log("Test Complete")

    def test_value_props(self, schematic: Schematic) -> None:

        self.log("Testing default properties of schematic value")
        value = schematic.create_value(f"{self.name}_uptime")
        default_props: dict[str, float | str | bool] = {
            "channel": f"{self.name}_uptime",
            "notation": "standard",
            "precision": 2,
            "averaging_window": 1,
            "stale_color": "#C29D0A",  # pluto-warning-m1
            "stale_timeout": 5,
        }
        schematic.assert_symbol_properties(value, default_props)

        self.log("Testing edited properties of schematic value")
        expected_edited_props: dict[str, float | str | bool] = {
            "channel": f"{self.name}_time",
            "notation": "scientific",
            "precision": 4,
            "averaging_window": 4,
            "stale_color": "#FF0000",
            "stale_timeout": 10,
        }
        value.edit_properties(
            channel_name=f"{self.name}_time",
            notation="scientific",
            precision=4,
            averaging_window=4,
            stale_color="#FF0000",
            stale_timeout=10,
        )
        schematic.assert_symbol_properties(value, expected_edited_props)
        value.delete()

        self.log("Testing new node with non-default properties")
        non_default_props: dict[str, float | str | bool] = {
            "channel": f"{self.name}_state",
            "notation": "engineering",
            "precision": 7,
            "averaging_window": 3,
            "stale_color": "#00FF00",
            "stale_timeout": 15,
        }
        non_default_value = schematic.create_value(
            f"{self.name}_state",
            notation="engineering",
            precision=7,
            averaging_window=3,
            stale_color="#00FF00",
            stale_timeout=15,
        )
        schematic.assert_symbol_properties(non_default_value, non_default_props)
        non_default_value.delete()

    def test_button_props(self, schematic: Schematic) -> None:
        client = self.client

        self.log("Testing default properties of schematic button")
        CHANNEL_NAME = "button_cmd"
        INDEX_NAME = "button_idx"

        index_ch = client.channels.create(
            name=INDEX_NAME,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        cmd_ch = client.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.UINT8,
            is_index=False,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        button = schematic.create_button(
            channel_name=CHANNEL_NAME,
        )
        default_props = button.get_properties()
        expected_default_props = {
            "channel": CHANNEL_NAME,
            "activation_delay": 0,
            "show_control_chip": True,
            "mode": "fire",
        }
        assert (
            default_props == expected_default_props
        ), f"Props mismatch!\nActual: {default_props}\nExpected: {expected_default_props}"

        self.log("Testing edited properties of schematic button")
        button.edit_properties(
            channel_name=CHANNEL_NAME,
            activation_delay=4.2,
            show_control_chip=False,
            mode="Momentary",
        )
        expected_edited_props: dict[str, float | str | bool] = {
            "channel": CHANNEL_NAME,
            "activation_delay": 4.2,
            "show_control_chip": False,
            "mode": "momentary",
        }
        schematic.assert_symbol_properties(button, expected_edited_props)
        button.delete()

        self.log("Testing non-default properties of schematic button")
        non_default_props: dict[str, float | str | bool] = {
            "channel": CHANNEL_NAME,
            "activation_delay": 2.3,
            "show_control_chip": True,
            "mode": "pulse",
        }
        non_default_button = schematic.create_button(
            channel_name=CHANNEL_NAME,
            activation_delay=2.3,
            show_control_chip=True,
            mode="pulse",
        )
        schematic.assert_symbol_properties(non_default_button, non_default_props)
        non_default_button.delete()
