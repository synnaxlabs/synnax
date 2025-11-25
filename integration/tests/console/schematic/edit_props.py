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
from console.schematic.schematic import PropertyDict, Schematic

CHANNEL_NAME = "button_cmd"
INDEX_NAME = "button_idx"


class EditProps(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        client = self.client

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

        schematic = Schematic(self.client, self.console, "edit_symbol_props")
        self.test_schematic_props(schematic)
        self.test_value_props(schematic)
        self.test_button_props(schematic)
        self.log("Test Complete")

    def test_schematic_props(self, schematic: Schematic) -> None:
        self.log("Test 0: Schematic Properties")

        self.log("0.1 Change Properties")
        schematic.assert_properties()
        schematic.edit_properties(control_authority=7)
        schematic.assert_properties(control_authority=7, show_control_legend=True)

        schematic.edit_properties(show_control_legend=False)
        schematic.assert_properties(control_authority=7, show_control_legend=False)

        schematic.edit_properties(control_authority=128, show_control_legend=True)
        schematic.assert_properties(control_authority=128, show_control_legend=True)

        self.log("0.2 Acquire Control")
        button = schematic.create.button(channel_name=CHANNEL_NAME)
        schematic.acquire_control()
        schematic.assert_control_status(True)
        schematic.assert_control_legend_visible(True)

        self.log("0.3 Hide Legend")
        schematic.release_control()
        schematic.assert_control_status(False)
        schematic.enable_edit()
        schematic.assert_edit_status(True)
        schematic.edit_properties(show_control_legend=False)
        schematic.acquire_control()
        schematic.assert_control_legend_visible(False)

        # Clean up schematic
        schematic.release_control()
        schematic.enable_edit()
        button.delete()

    def test_value_props(self, schematic: Schematic) -> None:
        self.log("Test 1: Value Properties")

        self.log("1.1 Default")
        value = schematic.create.value(f"{self.name}_uptime")
        default_props: PropertyDict = {
            "channel": f"{self.name}_uptime",
            "notation": "standard",
            "precision": 2,
            "averaging_window": 1,
            "stale_color": "#C29D0A",  # pluto-warning-m1
            "stale_timeout": 5,
        }
        schematic.assert_symbol_properties(value, default_props)

        self.log("1.2 Edited")
        expected_edited_props: PropertyDict = {
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

        self.log("1.3 Non-Default")
        non_default_props: PropertyDict = {
            "channel": f"{self.name}_state",
            "notation": "engineering",
            "precision": 7,
            "averaging_window": 3,
            "stale_color": "#00FF00",
            "stale_timeout": 15,
        }
        non_default_value = schematic.create.value(
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
        self.log("Test 2: Button Properties")

        self.log("2.1 Default")
        button = schematic.create.button(channel_name=CHANNEL_NAME)

        expected_default_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 0,
            "show_control_chip": True,
            "mode": "fire",
        }
        schematic.assert_symbol_properties(button, expected_default_props)

        self.log("2.2 Edited")
        button.edit_properties(
            channel_name=CHANNEL_NAME,
            activation_delay=4.2,
            show_control_chip=False,
            mode="Momentary",
        )
        expected_edited_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 4.2,
            "show_control_chip": False,
            "mode": "momentary",
        }
        schematic.assert_symbol_properties(button, expected_edited_props)
        button.delete()

        self.log("2.3 Non-Default")
        non_default_props: PropertyDict = {
            "channel": CHANNEL_NAME,
            "activation_delay": 2.3,
            "show_control_chip": True,
            "mode": "pulse",
        }
        non_default_button = schematic.create.button(
            channel_name=CHANNEL_NAME,
            activation_delay=2.3,
            show_control_chip=True,
            mode="pulse",
        )
        schematic.assert_symbol_properties(non_default_button, non_default_props)
        non_default_button.delete()
